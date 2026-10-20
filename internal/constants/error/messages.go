package error

const (
	InvalidRequestMessage      = "invalid request"
	UnauthorizedMessage        = "unauthorized"
	ForbiddenMessage           = "forbidden"
	NotFoundMessage            = "resource not found"
	ConflictMessage            = "request conflicts with current state"
	RateLimitExceededMessage   = "rate limit exceeded"
	ServiceUnavailableMessage  = "service temporarily unavailable"
	InternalServerErrorMessage = "internal server error"

	AuthEmailAlreadyRegisteredMessage  = "email is already registered"
	AuthInvalidCredentialsMessage      = "invalid credentials"
	AuthRefreshTokenReusedMessage      = "refresh token is invalid or has already been used"
	AuthUserNotFoundMessage            = "user not found"
	AuthNoAssignedRolesMessage         = "user authorization is not configured"
	AuthMissingAuthorizationMessage    = "missing authorization header"
	AuthInvalidAuthorizationMessage    = "invalid authorization header"
	AuthInvalidTokenMessage            = "invalid token"
	AuthSessionInvalidMessage          = "invalid or revoked session"
	AuthMissingContextMessage          = "missing auth context"
	AuthInvalidContextMessage          = "invalid auth context"
	AuthInsufficientPermissionsMessage = "insufficient permissions"
)
