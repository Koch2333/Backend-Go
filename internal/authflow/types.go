package authflow

import "time"

// Store provides persistence for TOTP and passkey state.
type Store interface {
	GetTOTP(username string) (secret string, enabled bool, err error)
	SetTOTP(username, secret string, enabled bool) error

	GetCredentials(username string) ([]Credential, error)
	SaveCredential(c *Credential) error
	DeleteCredential(username, credID string) error
	UpdateCounter(credID string, counter uint32) error
	ListCredentials(username string) ([]CredentialInfo, error)
}

// Credential is a stored WebAuthn passkey.
type Credential struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	PublicKey []byte    `json:"publicKey"`
	Counter   uint32    `json:"counter"`
	CreatedAt time.Time `json:"createdAt"`
}

// CredentialInfo is the public summary of a Credential.
type CredentialInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// Config holds all settings for a Flow.
type Config struct {
	Store             Store
	AdminUsername     string
	AdminPasswordHash string
	JWTSecret         []byte
	JWTTTL            time.Duration
	TOTPIssuer        string
	WebAuthnRPID      string
	WebAuthnRPName    string
	WebAuthnOrigins   []string
}
