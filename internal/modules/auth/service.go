package auth

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"gameco-technical-assessment/internal/config"
	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/constants/role"
	"gameco-technical-assessment/internal/platform/audit"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/identifier"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
	redis      *redis.Client
	jwtManager *platformauth.JWTManager
	config     config.AppConfig
	clock      platformclock.Clock
}

type SessionMetadata struct {
	UserAgent string
	IPAddress string
}

const (
	maxLoginFailures  = 5
	loginLockDuration = 15 * time.Minute
	loginLockReason   = "failed_attempts"
)

var dummyPasswordHash = func() string {
	raw := make([]byte, 32)
	if _, err := cryptorand.Read(raw); err != nil {
		panic(err)
	}
	hash, err := platformauth.HashPassword(string(raw))
	if err != nil {
		panic(err)
	}
	return hash
}()

func NewService(repository *Repository, redisClient *redis.Client, jwtManager *platformauth.JWTManager, cfg config.AppConfig) *Service {
	return NewServiceWithClock(repository, redisClient, jwtManager, cfg, platformclock.System{})
}

func NewServiceWithClock(repository *Repository, redisClient *redis.Client, jwtManager *platformauth.JWTManager, cfg config.AppConfig, timeSource platformclock.Clock) *Service {
	if timeSource == nil {
		timeSource = platformclock.System{}
	}
	return &Service{repository: repository, redis: redisClient, jwtManager: jwtManager, config: cfg, clock: timeSource}
}

func (s *Service) Register(ctx context.Context, input RegisterRequest, metadata SessionMetadata) (TokenResponse, error) {
	exists, err := s.repository.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return TokenResponse{}, err
	}
	if exists {
		return TokenResponse{}, faults.New(errorconst.AuthEmailAlreadyRegistered, nil)
	}
	passwordHash, err := platformauth.HashPassword(input.Password)
	if err != nil {
		return TokenResponse{}, err
	}
	publicID, err := identifier.NewPublicID()
	if err != nil {
		return TokenResponse{}, err
	}
	var user User
	err = s.repository.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err = s.repository.Create(ctx, tx, User{PublicID: publicID, Email: strings.ToLower(strings.TrimSpace(input.Email)), PasswordHash: passwordHash})
		if err != nil {
			return err
		}
		if err := s.repository.AssignRole(ctx, tx, user.ID, role.Buyer); err != nil {
			return err
		}
		return s.repository.AssignRole(ctx, tx, user.ID, role.Seller)
	})
	if err != nil {
		return TokenResponse{}, err
	}
	return s.issueTokenPair(ctx, user, metadata)
}

func (s *Service) Login(ctx context.Context, input LoginRequest, metadata SessionMetadata) (TokenResponse, error) {
	user, err := s.repository.FindByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = platformauth.ComparePassword(dummyPasswordHash, input.Password)
			s.recordFailedLoginAudit(ctx, input.Email, nil, "invalid_credentials")
			return TokenResponse{}, faults.New(errorconst.AuthInvalidCredentials, nil)
		}
		return TokenResponse{}, err
	}

	securityState, stateErr := s.repository.FindLoginSecurityState(ctx, user.ID)
	if stateErr != nil && !errors.Is(stateErr, gorm.ErrRecordNotFound) {
		return TokenResponse{}, stateErr
	}
	if securityState.LockedUntil != nil && s.clock.Now().Before(*securityState.LockedUntil) {
		s.recordFailedLoginAudit(ctx, input.Email, &user.ID, "account_locked")
		return TokenResponse{}, faults.New(errorconst.AuthInvalidCredentials, nil)
	}

	if err := platformauth.ComparePassword(user.PasswordHash, input.Password); err != nil {
		if recordErr := s.repository.RecordLoginFailure(ctx, user.ID, s.clock.Now(), s.clock.Now().Add(loginLockDuration)); recordErr != nil {
			return TokenResponse{}, recordErr
		}
		s.recordFailedLoginAudit(ctx, input.Email, &user.ID, "invalid_credentials")
		return TokenResponse{}, faults.New(errorconst.AuthInvalidCredentials, err)
	}
	if err := s.repository.ResetLoginSecurityState(ctx, user.ID); err != nil {
		return TokenResponse{}, err
	}
	return s.issueTokenPair(ctx, user, metadata)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, metadata SessionMetadata) (TokenResponse, error) {
	refreshHash := platformauth.HashToken(refreshToken)
	session, err := s.repository.FindActiveSessionByRefreshHash(ctx, refreshHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reuseErr := s.repository.HandleRefreshTokenReuse(ctx, refreshHash)
			if errors.Is(reuseErr, ErrRefreshTokenReuse) {
				return TokenResponse{}, faults.New(errorconst.AuthRefreshTokenReused, reuseErr)
			}
			return TokenResponse{}, reuseErr
		}
		return TokenResponse{}, err
	}
	user, err := s.repository.FindByID(ctx, session.UserID)
	if err != nil {
		return TokenResponse{}, err
	}
	access, err := s.repository.Access(ctx, user.ID)
	if err != nil {
		return TokenResponse{}, err
	}
	jtiID, err := identifier.NewPublicID()
	if err != nil {
		return TokenResponse{}, err
	}
	jti := jtiID.String()
	accessToken, err := s.jwtManager.GenerateAccessToken(fmt.Sprintf("%d", user.ID), jti, s.config.AccessTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}
	newRefreshToken, err := platformauth.GenerateRefreshToken()
	if err != nil {
		return TokenResponse{}, err
	}
	newRefreshHash := platformauth.HashToken(newRefreshToken)
	replacement := AuthSession{
		JTI: jti, RefreshHash: newRefreshHash,
		Roles: access.Roles, Permissions: access.Permissions,
		UserAgent: metadata.UserAgent, IPAddress: metadata.IPAddress,
		ExpiresAt: s.clock.Now().Add(s.config.RefreshTokenTTL),
	}
	if _, err := s.repository.RotateRefreshSession(ctx, refreshHash, replacement); err != nil {
		if errors.Is(err, ErrRefreshTokenReuse) {
			return TokenResponse{}, faults.New(errorconst.AuthRefreshTokenReused, err)
		}
		return TokenResponse{}, err
	}
	_ = s.redis.Del(ctx, s.refreshSessionKey(refreshHash)).Err()
	_ = s.redis.Set(ctx, s.refreshSessionKey(newRefreshHash), jti, s.config.RefreshTokenTTL).Err()
	return TokenResponse{AccessToken: accessToken, RefreshToken: newRefreshToken, TokenType: "Bearer", ExpiresIn: int64(s.config.AccessTokenTTL.Seconds()), User: mapUser(user, access)}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshHash := platformauth.HashToken(refreshToken)
	if err := s.repository.RevokeSessionByRefreshHash(ctx, refreshHash); err != nil {
		return err
	}
	return s.redis.Del(ctx, s.refreshSessionKey(refreshHash)).Err()
}

func (s *Service) Me(ctx context.Context, userID uint) (UserResponse, error) {
	user, err := s.repository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserResponse{}, faults.New(errorconst.AuthUserNotFound, err)
		}
		return UserResponse{}, err
	}
	access, err := s.repository.Access(ctx, user.ID)
	if err != nil {
		return UserResponse{}, err
	}
	return mapUser(user, access), nil
}

func (s *Service) ValidateSession(ctx context.Context, jti string) (Access, error) {
	session, err := s.repository.FindActiveSessionByJTI(ctx, jti)
	if err != nil {
		return Access{}, err
	}
	return Access{UserID: session.UserID, Roles: session.Roles, Permissions: session.Permissions}, nil
}

func (s *Service) issueTokenPair(ctx context.Context, user User, metadata SessionMetadata) (TokenResponse, error) {
	access, err := s.repository.Access(ctx, user.ID)
	if err != nil {
		return TokenResponse{}, err
	}
	if len(access.Roles) == 0 {
		return TokenResponse{}, faults.New(errorconst.AuthNoAssignedRoles, nil)
	}
	jtiID, err := identifier.NewPublicID()
	if err != nil {
		return TokenResponse{}, err
	}
	jti := jtiID.String()
	accessToken, err := s.jwtManager.GenerateAccessToken(fmt.Sprintf("%d", user.ID), jti, s.config.AccessTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}
	refreshToken, err := platformauth.GenerateRefreshToken()
	if err != nil {
		return TokenResponse{}, err
	}
	refreshHash := platformauth.HashToken(refreshToken)
	if _, err := s.repository.CreateSession(ctx, AuthSession{
		UserID: user.ID, JTI: jti, RefreshHash: refreshHash,
		Roles: access.Roles, Permissions: access.Permissions,
		UserAgent: metadata.UserAgent, IPAddress: metadata.IPAddress,
		ExpiresAt: s.clock.Now().Add(s.config.RefreshTokenTTL),
	}); err != nil {
		return TokenResponse{}, err
	}
	_ = s.redis.Set(ctx, s.refreshSessionKey(refreshHash), jti, s.config.RefreshTokenTTL).Err()
	return TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresIn: int64(s.config.AccessTokenTTL.Seconds()), User: mapUser(user, access)}, nil
}

func (s *Service) recordFailedLoginAudit(ctx context.Context, email string, actorUserID *uint, reason string) {
	_ = audit.Write(ctx, s.repository.DB(), audit.Event{
		ActorUserID:   actorUserID,
		EventType:     "auth.login_failed",
		AggregateType: "login_attempt",
		AggregateID:   platformauth.HashToken(strings.ToLower(strings.TrimSpace(email))),
		Metadata:      map[string]any{"reason": reason},
	})
}

func (s *Service) refreshSessionKey(refreshHash string) string {
	return "auth:refresh:" + refreshHash
}

func mapUser(user User, access Access) UserResponse {
	return UserResponse{PublicID: user.PublicID, Email: user.Email, Roles: access.Roles, Permissions: access.Permissions}
}
