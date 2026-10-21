package middleware

import (
	"context"
	"strconv"
	"strings"

	errorconst "gameco-technical-assessment/internal/constants/error"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const (
	ClaimsKey = "claims"
	ActorKey  = "actor"
)

type ActorResolver func(context.Context, string) (platformauth.Actor, error)

func Auth(jwtManager *platformauth.JWTManager, resolve ActorResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httpx.HandleError(c, faults.New(errorconst.AuthMissingAuthorization, nil))
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == header {
			httpx.HandleError(c, faults.New(errorconst.AuthInvalidAuthorization, nil))
			c.Abort()
			return
		}
		claims, err := jwtManager.ParseAccessToken(tokenString)
		if err != nil || claims.TokenID == "" {
			httpx.HandleError(c, faults.New(errorconst.AuthInvalidToken, err))
			c.Abort()
			return
		}
		actor, err := resolve(c.Request.Context(), claims.TokenID)
		if err != nil {
			httpx.HandleError(c, faults.New(errorconst.AuthSessionInvalid, err))
			c.Abort()
			return
		}
		if actor.UserID == 0 || claims.UserID != strconv.FormatUint(uint64(actor.UserID), 10) {
			httpx.HandleError(c, faults.New(errorconst.AuthInvalidContext, nil))
			c.Abort()
			return
		}
		c.Set(ClaimsKey, claims)
		c.Set(ActorKey, actor)
		c.Next()
	}
}

func RequirePermissions(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(ActorKey)
		if !ok {
			httpx.HandleError(c, faults.New(errorconst.AuthMissingContext, nil))
			c.Abort()
			return
		}
		actor, ok := value.(platformauth.Actor)
		if !ok {
			httpx.HandleError(c, faults.New(errorconst.AuthInvalidContext, nil))
			c.Abort()
			return
		}
		allowed := make(map[string]struct{}, len(actor.Permissions))
		for _, permission := range actor.Permissions {
			allowed[permission] = struct{}{}
		}
		for _, permission := range required {
			if _, ok := allowed[permission]; !ok {
				httpx.HandleError(c, faults.New(errorconst.AuthInsufficientPermissions, nil))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func RequireRoles(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get(ActorKey)
		if !ok {
			httpx.HandleError(c, faults.New(errorconst.AuthMissingContext, nil))
			c.Abort()
			return
		}
		actor, ok := value.(platformauth.Actor)
		if !ok {
			httpx.HandleError(c, faults.New(errorconst.AuthInvalidContext, nil))
			c.Abort()
			return
		}
		roles := make(map[string]struct{}, len(actor.Roles))
		for _, role := range actor.Roles {
			roles[role] = struct{}{}
		}
		for _, requiredRole := range required {
			if _, ok := roles[requiredRole]; ok {
				c.Next()
				return
			}
		}
		httpx.HandleError(c, faults.New(errorconst.AuthInsufficientPermissions, nil))
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

func ActorFromContext(c *gin.Context) (platformauth.Actor, bool) {
	value, ok := c.Get(ActorKey)
	if !ok {
		return platformauth.Actor{}, false
	}
	actor, ok := value.(platformauth.Actor)
	return actor, ok
}
