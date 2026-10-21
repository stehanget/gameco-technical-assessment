package faults

type Kind string

const (
	KindInvalidInput    Kind = "invalid_input"
	KindUnauthenticated Kind = "unauthenticated"
	KindForbidden       Kind = "forbidden"
	KindNotFound        Kind = "not_found"
	KindConflict        Kind = "conflict"
	KindRateLimit       Kind = "rate_limit"
	KindUnavailable     Kind = "unavailable"
	KindInternal        Kind = "internal"
)

type Definition struct {
	Code    string
	Kind    Kind
	Message string
}

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Cause   error
}

func New(definition Definition, cause error) *Error {
	return &Error{
		Kind:    definition.Kind,
		Code:    definition.Code,
		Message: definition.Message,
		Cause:   cause,
	}
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Code
	}
	return e.Code + ": " + e.Cause.Error()
}

func (e *Error) Unwrap() error {
	return e.Cause
}
