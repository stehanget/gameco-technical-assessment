package auth

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }
func (r *Repository) DB() *gorm.DB          { return r.db }

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
	return session, nil
}

func (r *Repository) FindActiveSessionByRefreshHash(ctx context.Context, refreshHash string) (AuthSession, error) {
	var session AuthSession
	err := r.db.WithContext(ctx).
		Where("refresh_hash = ? AND revoked_at IS NULL AND expires_at > ?", refreshHash, time.Now()).
		First(&session).Error
	return session, err
}

func (r *Repository) FindActiveSessionByJTI(ctx context.Context, jti string) (AuthSession, error) {
	var session AuthSession
	err := r.db.WithContext(ctx).
		Where("jti = ? AND revoked_at IS NULL AND expires_at > ?", jti, time.Now()).
		First(&session).Error
	return session, err
}

func (r *Repository) UpdateSession(ctx context.Context, session AuthSession) error {
	return r.db.WithContext(ctx).Save(&session).Error
}

func (r *Repository) RevokeSessionByRefreshHash(ctx context.Context, refreshHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&AuthSession{}).
		Where("refresh_hash = ? AND revoked_at IS NULL", refreshHash).
		Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error
}
