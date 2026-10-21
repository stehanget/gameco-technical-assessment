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

	CatalogInvalidCursorMessage   = "invalid catalog cursor"
	CatalogInvalidSortMessage     = "invalid catalog sorting"
	CatalogProductNotFoundMessage = "product not found"
)

func MessageForCode(code string) string {
	switch code {
	case AuthEmailAlreadyRegistered:
		return AuthEmailAlreadyRegisteredMessage
	case AuthInvalidCredentials:
		return AuthInvalidCredentialsMessage
	case AuthRefreshTokenReused:
		return AuthRefreshTokenReusedMessage
	case AuthUserNotFound:
		return AuthUserNotFoundMessage
	case AuthNoAssignedRoles:
		return AuthNoAssignedRolesMessage
	case AuthMissingAuthorization:
		return AuthMissingAuthorizationMessage
	case AuthInvalidAuthorization:
		return AuthInvalidAuthorizationMessage
	case AuthInvalidToken:
		return AuthInvalidTokenMessage
	case AuthSessionInvalid:
		return AuthSessionInvalidMessage
	case AuthMissingContext:
		return AuthMissingContextMessage
	case AuthInvalidContext:
		return AuthInvalidContextMessage
	case AuthInsufficientPermissions:
		return AuthInsufficientPermissionsMessage
	case CatalogInvalidCursor:
		return CatalogInvalidCursorMessage
	case CatalogInvalidSort:
		return CatalogInvalidSortMessage
	case CatalogProductNotFound:
		return CatalogProductNotFoundMessage
	case InvalidRequest:
		return InvalidRequestMessage
	case Unauthorized:
		return UnauthorizedMessage
	case Forbidden:
		return ForbiddenMessage
	case NotFound:
		return NotFoundMessage
	case Conflict:
		return ConflictMessage
	case RateLimitExceeded:
		return RateLimitExceededMessage
	case ServiceUnavailable:
		return ServiceUnavailableMessage
	case InternalServerError:
		return InternalServerErrorMessage
	default:
		return ""
	}
}
