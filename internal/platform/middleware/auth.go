package middleware

import (
	"context"
	"strings"

	errorconst "gameco-technical-assessment/internal/constants/error"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const (
	ClaimsKey = "claims"
	AccessKey = "access"
)

type SessionResolver func(context.Context, string) ([]string, []string, error)

type Access struct {
	Roles       []string
	Permissions []string
}

func Auth(jwtManager *platformauth.JWTManager, resolve SessionResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthMissingAuthorization, nil))
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == header {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidAuthorization, nil))
			c.Abort()
			return
		}
		claims, err := jwtManager.ParseAccessToken(tokenString)
		if err != nil || claims.TokenID == "" {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidToken, err))
			c.Abort()
			return
		}
		roles, permissions, err := resolve(c.Request.Context(), claims.TokenID)
		if err != nil {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthSessionInvalid, err))
			c.Abort()
			return
		}
		c.Set(ClaimsKey, claims)
		c.Set(AccessKey, Access{Roles: roles, Permissions: permissions})
		c.Next()
	}
}

func RequirePermissions(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(AccessKey)
		if !ok {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthMissingContext, nil))
			c.Abort()
			return
		}
		access, ok := value.(Access)
		if !ok {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidContext, nil))
			c.Abort()
			return
		}
		allowed := make(map[string]struct{}, len(access.Permissions))
		for _, permission := range access.Permissions {
			allowed[permission] = struct{}{}
		}
		for _, permission := range required {
			if _, ok := allowed[permission]; !ok {
				httpx.HandleError(c, faults.New(faults.KindForbidden, errorconst.AuthInsufficientPermissions, nil))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func RequireRoles(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(AccessKey)
		if !ok {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthMissingContext, nil))
			c.Abort()
			return
		}
		access, ok := value.(Access)
		if !ok {
			httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidContext, nil))
			c.Abort()
			return
		}
		roles := make(map[string]struct{}, len(access.Roles))
		for _, role := range access.Roles {
			roles[role] = struct{}{}
		}
		for _, requiredRole := range required {
			if _, ok := roles[requiredRole]; ok {
				c.Next()
				return
			}
		}
		httpx.HandleError(c, faults.New(faults.KindForbidden, errorconst.AuthInsufficientPermissions, nil))
		c.Abort()
	}
}

func ClaimsFromContext(c *gin.Context) (*platformauth.AccessTokenClaims, bool) {
	claims, ok := c.Get(ClaimsKey)
	if !ok {
		return nil, false
	}
	value, ok := claims.(*platformauth.AccessTokenClaims)
	return value, ok
}
