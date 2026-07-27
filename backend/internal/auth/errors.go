package auth

// ValidationError indicates the request input failed validation rules.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ConflictError indicates a resource already exists (e.g. duplicate email).
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// AuthError indicates authentication failed (bad credentials).
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
