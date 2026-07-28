package service

// ValidationError indicates the request input failed validation rules.
// Handlers should map this to HTTP 400.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NotFoundError indicates requested resource does not exist.
// Handlers should map this to HTTP 404.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// ForbiddenError indicates the user is authenticated but not allowed
// to perform this action (e.g. not the owner). Handlers should map
// this to HTTP 403.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return e.Message
}

// ConflictError indicates the request is well-formed and the caller is allowed,
// but it clashes with the order's CURRENT state - e.g. paying an order that's
// still 'pending', or confirming one that's already 'cancelled'. Handlers should
// map this to HTTP 409.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}