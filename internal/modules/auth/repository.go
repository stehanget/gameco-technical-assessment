package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRefreshTokenReuse = errors.New("invalid or reused refresh token")

type Repository struct {
	db    *gorm.DB
	redis redis.UniversalClient
}

func NewRepository(db *gorm.DB, redisClient redis.UniversalClient) *Repository {
	return &Repository{db: db, redis: redisClient}
}

func (r *Repository) DB() *gorm.DB { return r.db }

func (r *Repository) Create(ctx context.Context, db *gorm.DB, user User) (User, error) {
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error; err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) FindByID(ctx context.Context, userID uint) (User, error) {
	var user User
	if err := r.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := r.FindByEmail(ctx, email)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func (r *Repository) Access(ctx context.Context, userID uint) (Access, error) {
	type accessRow struct {
		RoleCode       string
		PermissionCode string
	}
	var rows []accessRow
	if err := r.db.WithContext(ctx).Table("user_roles AS ur").
		Select("r.code AS role_code, p.code AS permission_code").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Joins("JOIN role_permissions rp ON rp.role_id = r.id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("ur.user_id = ?", userID).
		Order("r.code ASC, p.code ASC").
		Scan(&rows).Error; err != nil {
		return Access{}, err
	}

	roles := make([]string, 0)
	permissions := make([]string, 0)
	seenRoles := map[string]struct{}{}
	seenPermissions := map[string]struct{}{}
	for _, row := range rows {
		if _, ok := seenRoles[row.RoleCode]; !ok {
			seenRoles[row.RoleCode] = struct{}{}
			roles = append(roles, row.RoleCode)
		}
		if _, ok := seenPermissions[row.PermissionCode]; !ok {
			seenPermissions[row.PermissionCode] = struct{}{}
			permissions = append(permissions, row.PermissionCode)
		}
	}
	sort.Strings(roles)
	sort.Strings(permissions)
	return Access{Roles: roles, Permissions: permissions}, nil
}

func (r *Repository) AssignRole(ctx context.Context, db *gorm.DB, userID uint, roleCode string) error {
	var role Role
	if err := db.WithContext(ctx).Where("code = ?", roleCode).First(&role).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, role.ID).FirstOrCreate(&UserRole{UserID: userID, RoleID: role.ID}).Error
}

func (r *Repository) CreateSession(ctx context.Context, session AuthSession) (AuthSession, error) {
	if err := r.db.WithContext(ctx).Create(&session).Error; err != nil {
		return AuthSession{}, err
	}
	r.cacheSession(ctx, session)
	return session, nil
}

func (r *Repository) FindActiveSessionByRefreshHash(ctx context.Context, refreshHash string) (AuthSession, error) {
	var session AuthSession
	err := r.db.WithContext(ctx).
		Where("refresh_hash = ? AND revoked_at IS NULL AND expires_at > ?", refreshHash, time.Now()).
		First(&session).Error
	if err == nil {
		r.cacheSession(ctx, session)
	}
	return session, err
}

func (r *Repository) FindActiveSessionByJTI(ctx context.Context, jti string) (AuthSession, error) {
	if cached, ok := r.cachedActiveSession(ctx, jti); ok {
		return cached, nil
	}

	var session AuthSession
	err := r.db.WithContext(ctx).
		Where("jti = ? AND revoked_at IS NULL AND expires_at > ?", jti, time.Now()).
		First(&session).Error
	if err == nil {
		r.cacheSession(ctx, session)
	}
	return session, err
}

func (r *Repository) UpdateSession(ctx context.Context, session AuthSession) error {
	if err := r.db.WithContext(ctx).Save(&session).Error; err != nil {
		return err
	}
	r.cacheSession(ctx, session)
	return nil
}

func (r *Repository) RotateRefreshSession(ctx context.Context, refreshHash string, replacement AuthSession) (AuthSession, error) {
	var oldSession AuthSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("refresh_hash = ? AND revoked_at IS NULL AND expires_at > ?", refreshHash, time.Now()).
			First(&oldSession).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				var reused AuthSession
				if lookupErr := tx.Where("refresh_hash = ?", refreshHash).First(&reused).Error; lookupErr == nil && reused.RevokedAt != nil {
					now := time.Now()
					if result := tx.Model(&AuthSession{}).Where("user_id = ? AND revoked_at IS NULL", reused.UserID).
						Updates(map[string]any{"revoked_at": now, "updated_at": now}); result.Error != nil {
						return result.Error
					}
					return ErrRefreshTokenReuse
				}
			}
			return ErrRefreshTokenReuse
		}

		now := time.Now()
		r.invalidateSession(ctx, oldSession)
		if result := tx.Model(&AuthSession{}).Where("id = ? AND revoked_at IS NULL", oldSession.ID).
			Updates(map[string]any{"revoked_at": now, "last_used_at": now, "updated_at": now}); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return ErrRefreshTokenReuse
		}

		replacement.UserID = oldSession.UserID
		return tx.Create(&replacement).Error
	})
	if err != nil {
		if errors.Is(err, ErrRefreshTokenReuse) {
			r.invalidateSession(ctx, oldSession)
		}
		return AuthSession{}, err
	}

	r.invalidateSession(ctx, oldSession)
	r.cacheSession(ctx, replacement)
	return oldSession, nil
}

func (r *Repository) RevokeSessionByRefreshHash(ctx context.Context, refreshHash string) error {
	var session AuthSession
	if err := r.db.WithContext(ctx).Where("refresh_hash = ? AND revoked_at IS NULL", refreshHash).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	r.invalidateSession(ctx, session)
	now := time.Now()
	return r.db.WithContext(ctx).Model(&AuthSession{}).
		Where("id = ? AND revoked_at IS NULL", session.ID).
		Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error
}

func (r *Repository) HandleRefreshTokenReuse(ctx context.Context, refreshHash string) error {
	var matched AuthSession
	if err := r.db.WithContext(ctx).Where("refresh_hash = ?", refreshHash).First(&matched).Error; err != nil {
		return ErrRefreshTokenReuse
	}

	var sessions []AuthSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL", matched.UserID).Find(&sessions).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&AuthSession{}).Where("user_id = ? AND revoked_at IS NULL", matched.UserID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error
	})
	if err != nil {
		return err
	}
	for _, session := range sessions {
		r.invalidateSession(ctx, session)
	}
	r.invalidateSession(ctx, matched)
	return ErrRefreshTokenReuse
}

const (
	sessionCacheTTL      = 30 * time.Second
	sessionCachePrefix   = "gameco:auth-session:"
	sessionRevokedPrefix = "gameco:auth-session-revoked:"
)

func (r *Repository) cachedActiveSession(ctx context.Context, jti string) (AuthSession, bool) {
	if r.redis == nil {
		return AuthSession{}, false
	}
	if _, err := r.redis.Get(ctx, sessionRevokedPrefix+jti).Result(); err == nil {
		return AuthSession{}, false
	}
	payload, err := r.redis.Get(ctx, sessionCachePrefix+jti).Bytes()
	if err != nil {
		return AuthSession{}, false
	}
	var session AuthSession
	if json.Unmarshal(payload, &session) != nil || session.RevokedAt != nil || !session.ExpiresAt.After(time.Now()) {
		return AuthSession{}, false
	}
	return session, true
}

func (r *Repository) cacheSession(ctx context.Context, session AuthSession) {
	if r.redis == nil || session.RevokedAt != nil {
		return
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return
	}
	if ttl > sessionCacheTTL {
		ttl = sessionCacheTTL
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return
	}
	_ = r.redis.Set(ctx, sessionCachePrefix+session.JTI, payload, ttl).Err()
}

func (r *Repository) invalidateSession(ctx context.Context, session AuthSession) {
	if r.redis == nil || session.JTI == "" {
		return
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	pipe := r.redis.Pipeline()
	pipe.Set(ctx, sessionRevokedPrefix+session.JTI, "1", ttl)
	pipe.Del(ctx, sessionCachePrefix+session.JTI)
	_, _ = pipe.Exec(ctx)
}
