package auth

// EmailSender defines the interface for sending emails (OTP codes, notifications, etc.)
type EmailSender interface {
	SendOTP(email, code string) error
}

// UserStore defines the interface for storing and retrieving user information for OTP auth
type UserStore interface {
	GetUserByEmail(email string) (*AuthUser, error)
	GetUserByUsername(username string) (*AuthUser, error)
}

// WithOTPAuth creates an AuthConfig that uses One-Time Password authentication via email
// TODO: Implement OTP authentication in future version
func WithOTPAuth(emailSender EmailSender, userStore UserStore) AuthConfig {
	panic("OTP authentication not implemented yet - use WithNoAuth() or WithBasicAuth() instead")
}
