package cart

const (
	StatusActive     = "active"
	StatusCheckedOut = "checked_out"
	StatusAbandoned  = "abandoned"

	AvailabilityAvailable   = "available"
	AvailabilityUnavailable = "unavailable"

	UnavailableReasonProductInactive   = "product_inactive"
	UnavailableReasonProductDeleted    = "product_deleted"
	UnavailableReasonInsufficientStock = "insufficient_stock"
)
