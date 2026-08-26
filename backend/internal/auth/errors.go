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

// AccountExistsError indicates an OAuth sign-in matched an account that has a
// password, which has to be proven before the identity can be linked.
type AccountExistsError struct {
	Message string
}

func (e *AccountExistsError) Error() string {
	return e.Message
}

// RetryError indicates a transient conflict; the same request may succeed.
type RetryError struct {
	Message string
}

func (e *RetryError) Error() string {
	return e.Message
}
