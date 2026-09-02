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

// ForbiddenError indicates the caller is authenticated but failed a check that
// re-proves who they are - today, the current password on a password change.
//
// Not AuthError, and the difference is load-bearing rather than pedantic. 401
// means "your session is no good", and the frontend's response interceptor
// acts on it: it silently refreshes and replays the request. A 401 for a
// mistyped password would burn a second bcrypt on the replay, and if the
// refresh itself failed it would clear the token and sign the user out - for a
// typo. 403 says "the session is fine, this particular attempt is not", which
// no interceptor treats as a session problem.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
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

type SuspendedError struct {
	Message string
}

func (e *SuspendedError) Error() string {
	return e.Message
}
