package error

import "gameco-technical-assessment/internal/platform/faults"

var (
	InvalidRequest = faults.Definition{
		Code: "common.invalid_request", Kind: faults.KindInvalidInput, Message: "invalid request",
	}
	Unauthorized = faults.Definition{
		Code: "common.unauthorized", Kind: faults.KindUnauthenticated, Message: "unauthorized",
	}
	Forbidden = faults.Definition{
		Code: "common.forbidden", Kind: faults.KindForbidden, Message: "forbidden",
	}
	NotFound = faults.Definition{
		Code: "common.not_found", Kind: faults.KindNotFound, Message: "resource not found",
	}
	Conflict = faults.Definition{
		Code: "common.conflict", Kind: faults.KindConflict, Message: "request conflicts with current state",
	}
	RateLimitExceeded = faults.Definition{
		Code: "common.rate_limit_exceeded", Kind: faults.KindRateLimit, Message: "rate limit exceeded",
	}
	ServiceUnavailable = faults.Definition{
		Code: "common.service_unavailable", Kind: faults.KindUnavailable, Message: "service temporarily unavailable",
	}
	InternalServerError = faults.Definition{
		Code: "common.internal_server_error", Kind: faults.KindInternal, Message: "internal server error",
	}

	AuthEmailAlreadyRegistered = faults.Definition{
		Code: "auth.email_already_registered", Kind: faults.KindConflict, Message: "email is already registered",
	}
	AuthInvalidCredentials = faults.Definition{
		Code: "auth.invalid_credentials", Kind: faults.KindUnauthenticated, Message: "invalid credentials",
	}
	AuthRefreshTokenReused = faults.Definition{
		Code: "auth.refresh_token_reused", Kind: faults.KindUnauthenticated, Message: "refresh token is invalid or has already been used",
	}
	AuthUserNotFound = faults.Definition{
		Code: "auth.user_not_found", Kind: faults.KindNotFound, Message: "user not found",
	}
	AuthNoAssignedRoles = faults.Definition{
		Code: "auth.no_assigned_roles", Kind: faults.KindInternal, Message: "user authorization is not configured",
	}
	AuthMissingAuthorization = faults.Definition{
		Code: "auth.missing_authorization", Kind: faults.KindUnauthenticated, Message: "missing authorization header",
	}
	AuthInvalidAuthorization = faults.Definition{
		Code: "auth.invalid_authorization", Kind: faults.KindUnauthenticated, Message: "invalid authorization header",
	}
	AuthInvalidToken = faults.Definition{
		Code: "auth.invalid_token", Kind: faults.KindUnauthenticated, Message: "invalid token",
	}
	AuthSessionInvalid = faults.Definition{
		Code: "auth.session_invalid", Kind: faults.KindUnauthenticated, Message: "invalid or revoked session",
	}
	AuthMissingContext = faults.Definition{
		Code: "auth.missing_context", Kind: faults.KindUnauthenticated, Message: "missing auth context",
	}
	AuthInvalidContext = faults.Definition{
		Code: "auth.invalid_context", Kind: faults.KindUnauthenticated, Message: "invalid auth context",
	}
	AuthInsufficientPermissions = faults.Definition{
		Code: "auth.insufficient_permissions", Kind: faults.KindForbidden, Message: "insufficient permissions",
	}

	CatalogInvalidCursor = faults.Definition{
		Code: "catalog.invalid_cursor", Kind: faults.KindInvalidInput, Message: "invalid catalog cursor",
	}
	CatalogInvalidSort = faults.Definition{
		Code: "catalog.invalid_sort", Kind: faults.KindInvalidInput, Message: "invalid catalog sorting",
	}
	CatalogProductNotFound = faults.Definition{
		Code: "catalog.product_not_found", Kind: faults.KindNotFound, Message: "product not found",
	}

	CartProductNotFound = faults.Definition{
		Code: "cart.product_not_found", Kind: faults.KindNotFound, Message: "product not found",
	}
	CartInsufficientStock = faults.Definition{
		Code: "cart.insufficient_stock", Kind: faults.KindConflict, Message: "requested quantity exceeds available stock",
	}
	SelfPurchaseNotAllowed = faults.Definition{
		Code: "commerce.self_purchase_not_allowed", Kind: faults.KindConflict, Message: "you cannot purchase your own product",
	}
	CheckoutIdempotencyKeyRequired = faults.Definition{
		Code: "checkout.idempotency_key_required", Kind: faults.KindInvalidInput, Message: "idempotency key is required",
	}
	CheckoutIdempotencyConflict = faults.Definition{
		Code: "checkout.idempotency_conflict", Kind: faults.KindConflict, Message: "idempotency key was used with a different request",
	}
	CheckoutEmptyCart = faults.Definition{
		Code: "checkout.empty_cart", Kind: faults.KindConflict, Message: "cart is empty",
	}
	CheckoutCartUnavailable = faults.Definition{
		Code: "checkout.cart_unavailable", Kind: faults.KindConflict, Message: "cart contains unavailable items",
	}
	CheckoutInsufficientStock = faults.Definition{
		Code: "checkout.insufficient_stock", Kind: faults.KindConflict, Message: "requested quantity exceeds available stock",
	}
	OrderNotFound = faults.Definition{
		Code: "order.not_found", Kind: faults.KindNotFound, Message: "order not found",
	}
)
