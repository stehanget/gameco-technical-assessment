package error

const (
	InvalidRequest      = "common.invalid_request"
	Unauthorized        = "common.unauthorized"
	Forbidden           = "common.forbidden"
	NotFound            = "common.not_found"
	Conflict            = "common.conflict"
	RateLimitExceeded   = "common.rate_limit_exceeded"
	ServiceUnavailable  = "common.service_unavailable"
	InternalServerError = "common.internal_server_error"

	AuthEmailAlreadyRegistered  = "auth.email_already_registered"
	AuthInvalidCredentials      = "auth.invalid_credentials"
	AuthRefreshTokenReused      = "auth.refresh_token_reused"
	AuthUserNotFound            = "auth.user_not_found"
	AuthNoAssignedRoles         = "auth.no_assigned_roles"
	AuthMissingAuthorization    = "auth.missing_authorization"
	AuthInvalidAuthorization    = "auth.invalid_authorization"
	AuthInvalidToken            = "auth.invalid_token"
	AuthSessionInvalid          = "auth.session_invalid"
	AuthMissingContext          = "auth.missing_context"
	AuthInvalidContext          = "auth.invalid_context"
	AuthInsufficientPermissions = "auth.insufficient_permissions"
)
