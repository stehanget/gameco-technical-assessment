package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gameco-technical-assessment/internal/config"
	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/constants/role"
	platformauth "gameco-technical-assessment/internal/platform/auth"
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
}

type SessionMetadata struct {
	UserAgent string
	IPAddress string
}

func NewService(repository *Repository, redisClient *redis.Client, jwtManager *platformauth.JWTManager, cfg config.AppConfig) *Service {
	return &Service{repository: repository, redis: redisClient, jwtManager: jwtManager, config: cfg}
}

func (s *Service) Register(ctx context.Context, input RegisterRequest, metadata SessionMetadata) (TokenResponse, error) {
	exists, err := s.repository.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return TokenResponse{}, err
	}
	if exists {
		return TokenResponse{}, faults.New(faults.KindConflict, errorconst.AuthEmailAlreadyRegistered, nil)
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
			return TokenResponse{}, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidCredentials, nil)
		}
		return TokenResponse{}, err
	}
	if err := platformauth.ComparePassword(user.PasswordHash, input.Password); err != nil {
		return TokenResponse{}, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidCredentials, err)
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
				return TokenResponse{}, faults.New(faults.KindUnauthenticated, errorconst.AuthRefreshTokenReused, reuseErr)
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
		ExpiresAt: time.Now().Add(s.config.RefreshTokenTTL),
	}
	if _, err := s.repository.RotateRefreshSession(ctx, refreshHash, replacement); err != nil {
		if errors.Is(err, ErrRefreshTokenReuse) {
			return TokenResponse{}, faults.New(faults.KindUnauthenticated, errorconst.AuthRefreshTokenReused, err)
		}
		return TokenResponse{}, err
	}
	_ = s.redis.Del(ctx, s.refreshSessionKey(refreshHash)).Err()
	if err := s.redis.Set(ctx, s.refreshSessionKey(newRefreshHash), jti, s.config.RefreshTokenTTL).Err(); err != nil {
		return TokenResponse{}, err
	}
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
			return UserResponse{}, faults.New(faults.KindNotFound, errorconst.AuthUserNotFound, err)
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
	return Access{Roles: session.Roles, Permissions: session.Permissions}, nil
}

func (s *Service) issueTokenPair(ctx context.Context, user User, metadata SessionMetadata) (TokenResponse, error) {
	access, err := s.repository.Access(ctx, user.ID)
	if err != nil {
		return TokenResponse{}, err
	}
	if len(access.Roles) == 0 {
		return TokenResponse{}, faults.New(faults.KindInternal, errorconst.AuthNoAssignedRoles, nil)
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
		ExpiresAt: time.Now().Add(s.config.RefreshTokenTTL),
	}); err != nil {
		return TokenResponse{}, err
	}
	if err := s.redis.Set(ctx, s.refreshSessionKey(refreshHash), jti, s.config.RefreshTokenTTL).Err(); err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresIn: int64(s.config.AccessTokenTTL.Seconds()), User: mapUser(user, access)}, nil
}

func (s *Service) refreshSessionKey(refreshHash string) string {
	return "auth:refresh:" + refreshHash
}

func mapUser(user User, access Access) UserResponse {
	return UserResponse{PublicID: user.PublicID, Email: user.Email, Roles: access.Roles, Permissions: access.Permissions}
}
