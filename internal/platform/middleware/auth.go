package middleware

import (
	"context"
	"net/http"
	"strings"

	platformauth "gameco-technical-assessment/internal/platform/auth"
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
			httpx.Error(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == header {
			httpx.Error(c, http.StatusUnauthorized, "invalid authorization header")
			c.Abort()
			return
		}
		claims, err := jwtManager.ParseAccessToken(tokenString)
		if err != nil || claims.TokenID == "" {
			httpx.Error(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}
		roles, permissions, err := resolve(c.Request.Context(), claims.TokenID)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "invalid or revoked session")
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
			httpx.Error(c, http.StatusUnauthorized, "missing auth context")
			c.Abort()
			return
		}
		access, ok := value.(Access)
		if !ok {
			httpx.Error(c, http.StatusUnauthorized, "invalid auth context")
			c.Abort()
			return
		}
		allowed := make(map[string]struct{}, len(access.Permissions))
		for _, permission := range access.Permissions {
			allowed[permission] = struct{}{}
		}
		for _, permission := range required {
			if _, ok := allowed[permission]; !ok {
				httpx.Error(c, http.StatusForbidden, "insufficient permissions")
				c.Abort()
				return
			}
		}
		c.Next()
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
