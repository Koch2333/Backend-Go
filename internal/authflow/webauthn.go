package authflow

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// ----- challenge sessions -----

type waChallenge struct {
	username  string
	challenge []byte
	expiresAt time.Time
}

type waPool struct {
	mu   sync.Mutex
	sess map[string]*waChallenge
}

func (p *waPool) store(c *waChallenge) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	id := base64.RawURLEncoding.EncodeToString(b)
	c.expiresAt = time.Now().Add(5 * time.Minute)
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for k, v := range p.sess {
		if now.After(v.expiresAt) {
			delete(p.sess, k)
		}
	}
	p.sess[id] = c
	return id
}

func (p *waPool) pop(id string) (*waChallenge, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.sess[id]
	if !ok {
		return nil, false
	}
	delete(p.sess, id)
	if time.Now().After(c.expiresAt) {
		return nil, false
	}
	return c, true
}

// ----- request/response types -----

// RegBeginOptions is sent to the browser to start a credential creation ceremony.
type RegBeginOptions struct {
	Challenge       string   `json:"challenge"`
	RPID            string   `json:"rpId"`
	RPName          string   `json:"rpName"`
	UserID          string   `json:"userId"`
	UserName        string   `json:"userName"`
	UserDisplayName string   `json:"userDisplayName"`
	Timeout         int      `json:"timeout"`
	ExcludeCredIDs  []string `json:"excludeCredentialIds"`
	SessionID       string   `json:"sessionId"`
}

// CredentialCreationResponse is what the browser sends after navigator.credentials.create().
type CredentialCreationResponse struct {
	ID       string              `json:"id"`
	RawID    string              `json:"rawId"`
	Type     string              `json:"type"`
	Response AttestationResponse `json:"response"`
}

// AttestationResponse holds the raw attestation data.
type AttestationResponse struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AttestationObject string `json:"attestationObject"`
}

// LoginBeginOptions is sent to the browser to start a credential assertion ceremony.
type LoginBeginOptions struct {
	Challenge        string           `json:"challenge"`
	RPID             string           `json:"rpId"`
	Timeout          int              `json:"timeout"`
	AllowCredentials []CredDescriptor `json:"allowCredentials"`
	SessionID        string           `json:"sessionId"`
}

// CredDescriptor identifies an allowed credential.
type CredDescriptor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// CredentialAssertionResponse is what the browser sends after navigator.credentials.get().
type CredentialAssertionResponse struct {
	ID       string            `json:"id"`
	RawID    string            `json:"rawId"`
	Type     string            `json:"type"`
	Response AssertionResponse `json:"response"`
}

// AssertionResponse holds the raw assertion data.
type AssertionResponse struct {
	ClientDataJSON    string `json:"clientDataJSON"`
	AuthenticatorData string `json:"authenticatorData"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}

// ----- ceremonies -----

func beginRegistration(pool *waPool, cfg *Config, username string, existingIDs [][]byte) (*RegBeginOptions, error) {
	chal := make([]byte, 32)
	if _, err := rand.Read(chal); err != nil {
		return nil, err
	}
	excludeIDs := make([]string, len(existingIDs))
	for i, id := range existingIDs {
		excludeIDs[i] = base64.RawURLEncoding.EncodeToString(id)
	}
	sessID := pool.store(&waChallenge{username: username, challenge: chal})
	return &RegBeginOptions{
		Challenge:       base64.RawURLEncoding.EncodeToString(chal),
		RPID:            cfg.WebAuthnRPID,
		RPName:          cfg.WebAuthnRPName,
		UserID:          base64.RawURLEncoding.EncodeToString([]byte(username)),
		UserName:        username,
		UserDisplayName: username,
		Timeout:         60000,
		ExcludeCredIDs:  excludeIDs,
		SessionID:       sessID,
	}, nil
}

func finishRegistration(pool *waPool, cfg *Config, sessionID string, resp *CredentialCreationResponse) (*Credential, error) {
	sess, ok := pool.pop(sessionID)
	if !ok {
		return nil, errors.New("webauthn: session expired")
	}
	cdJSON, err := waB64Decode(resp.Response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("webauthn: clientDataJSON: %w", err)
	}
	var cd waClientData
	if err := json.Unmarshal(cdJSON, &cd); err != nil {
		return nil, fmt.Errorf("webauthn: parse clientData: %w", err)
	}
	if cd.Type != "webauthn.create" {
		return nil, fmt.Errorf("webauthn: wrong type %q", cd.Type)
	}
	if err := waChallengeMatch(cd.Challenge, sess.challenge); err != nil {
		return nil, err
	}
	if err := waOriginAllowed(cd.Origin, cfg.WebAuthnOrigins); err != nil {
		return nil, err
	}
	attObjBytes, err := waB64Decode(resp.Response.AttestationObject)
	if err != nil {
		return nil, fmt.Errorf("webauthn: attestationObject: %w", err)
	}
	v, _, err := cborDecode(attObjBytes)
	if err != nil {
		return nil, fmt.Errorf("webauthn: attestationObject CBOR: %w", err)
	}
	attObj, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("webauthn: attestationObject not a map")
	}
	authDataRaw, ok := attObj["authData"].([]byte)
	if !ok {
		return nil, errors.New("webauthn: authData missing")
	}
	ad, err := parseAuthData(authDataRaw, true)
	if err != nil {
		return nil, fmt.Errorf("webauthn: authData: %w", err)
	}
	expected := sha256.Sum256([]byte(cfg.WebAuthnRPID))
	if string(ad.RPIDHash) != string(expected[:]) {
		return nil, errors.New("webauthn: rpIdHash mismatch")
	}
	return &Credential{
		ID:        base64.RawURLEncoding.EncodeToString(ad.CredentialID),
		Username:  sess.username,
		PublicKey: ad.CredentialPublicKey,
		Counter:   ad.SignCount,
		CreatedAt: time.Now(),
	}, nil
}

func beginLogin(pool *waPool, cfg *Config, username string, creds []Credential) (*LoginBeginOptions, error) {
	if len(creds) == 0 {
		return nil, errors.New("webauthn: no credentials registered")
	}
	chal := make([]byte, 32)
	if _, err := rand.Read(chal); err != nil {
		return nil, err
	}
	allow := make([]CredDescriptor, len(creds))
	for i, c := range creds {
		allow[i] = CredDescriptor{Type: "public-key", ID: c.ID}
	}
	sessID := pool.store(&waChallenge{username: username, challenge: chal})
	return &LoginBeginOptions{
		Challenge:        base64.RawURLEncoding.EncodeToString(chal),
		RPID:             cfg.WebAuthnRPID,
		Timeout:          60000,
		AllowCredentials: allow,
		SessionID:        sessID,
	}, nil
}

func finishLogin(pool *waPool, cfg *Config, sessionID string, resp *CredentialAssertionResponse, getCreds func(string) ([]Credential, error)) (*Credential, error) {
	sess, ok := pool.pop(sessionID)
	if !ok {
		return nil, errors.New("webauthn: session expired")
	}
	cdJSON, err := waB64Decode(resp.Response.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("webauthn: clientDataJSON: %w", err)
	}
	var cd waClientData
	if err := json.Unmarshal(cdJSON, &cd); err != nil {
		return nil, fmt.Errorf("webauthn: parse clientData: %w", err)
	}
	if cd.Type != "webauthn.get" {
		return nil, fmt.Errorf("webauthn: wrong type %q", cd.Type)
	}
	if err := waChallengeMatch(cd.Challenge, sess.challenge); err != nil {
		return nil, err
	}
	if err := waOriginAllowed(cd.Origin, cfg.WebAuthnOrigins); err != nil {
		return nil, err
	}
	authDataBytes, err := waB64Decode(resp.Response.AuthenticatorData)
	if err != nil {
		return nil, fmt.Errorf("webauthn: authenticatorData: %w", err)
	}
	ad, err := parseAuthData(authDataBytes, false)
	if err != nil {
		return nil, fmt.Errorf("webauthn: authData: %w", err)
	}
	expected := sha256.Sum256([]byte(cfg.WebAuthnRPID))
	if string(ad.RPIDHash) != string(expected[:]) {
		return nil, errors.New("webauthn: rpIdHash mismatch")
	}
	username := sess.username
	if username == "" {
		uh, _ := waB64Decode(resp.Response.UserHandle)
		username = string(uh)
	}
	creds, err := getCreds(username)
	if err != nil {
		return nil, fmt.Errorf("webauthn: get creds: %w", err)
	}
	credID := strings.TrimRight(resp.ID, "=")
	if credID == "" {
		credID = strings.TrimRight(resp.RawID, "=")
	}
	var matched *Credential
	for i := range creds {
		if creds[i].ID == credID {
			matched = &creds[i]
			break
		}
	}
	if matched == nil {
		return nil, errors.New("webauthn: credential not found")
	}
	pubKey, err := parseCOSEKey(matched.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("webauthn: parse pubkey: %w", err)
	}
	sig, err := waB64Decode(resp.Response.Signature)
	if err != nil {
		return nil, fmt.Errorf("webauthn: signature: %w", err)
	}
	cdHash := sha256.Sum256(cdJSON)
	verifyData := append(authDataBytes, cdHash[:]...)
	if err := waVerifySignature(pubKey, verifyData, sig); err != nil {
		return nil, fmt.Errorf("webauthn: %w", err)
	}
	matched.Counter = ad.SignCount
	return matched, nil
}

// ----- internals -----

type waClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Origin    string `json:"origin"`
}

type authDataParsed struct {
	RPIDHash            []byte
	Flags               byte
	SignCount           uint32
	CredentialID        []byte
	CredentialPublicKey []byte
}

func parseAuthData(data []byte, expectAT bool) (*authDataParsed, error) {
	if len(data) < 37 {
		return nil, errors.New("authData too short")
	}
	ad := &authDataParsed{
		RPIDHash:  data[:32],
		Flags:     data[32],
		SignCount: binary.BigEndian.Uint32(data[33:37]),
	}
	if expectAT {
		if ad.Flags&0x40 == 0 {
			return nil, errors.New("AT flag not set")
		}
		if len(data) < 55 {
			return nil, errors.New("authData too short for AT data")
		}
		credIDLen := int(binary.BigEndian.Uint16(data[53:55]))
		if len(data) < 55+credIDLen {
			return nil, errors.New("credentialId truncated")
		}
		ad.CredentialID = data[55 : 55+credIDLen]
		ad.CredentialPublicKey = data[55+credIDLen:]
	}
	return ad, nil
}

func parseCOSEKey(b []byte) (crypto.PublicKey, error) {
	v, _, err := cborDecode(b)
	if err != nil {
		return nil, fmt.Errorf("cose cbor: %w", err)
	}
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("cose: not a map")
	}
	ktyRaw, ok := m[uint64(1)]
	if !ok {
		return nil, errors.New("cose: missing kty")
	}
	switch ktyRaw {
	case uint64(2): // EC2
		xb, _ := m[int64(-2)].([]byte)
		yb, _ := m[int64(-3)].([]byte)
		if len(xb) == 0 || len(yb) == 0 {
			return nil, errors.New("cose: missing EC x/y")
		}
		var curve elliptic.Curve
		switch m[int64(-1)] {
		case uint64(1):
			curve = elliptic.P256()
		case uint64(2):
			curve = elliptic.P384()
		case uint64(3):
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("cose: unsupported crv %v", m[int64(-1)])
		}
		return &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)}, nil
	case uint64(3): // RSA
		nb, _ := m[int64(-1)].([]byte)
		eb, _ := m[int64(-2)].([]byte)
		if len(nb) == 0 || len(eb) == 0 {
			return nil, errors.New("cose: missing RSA n/e")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(new(big.Int).SetBytes(eb).Int64())}, nil
	}
	return nil, fmt.Errorf("cose: unsupported kty %v", ktyRaw)
}

type ecdsaSig struct{ R, S *big.Int }

func waVerifySignature(pub crypto.PublicKey, data, sig []byte) error {
	hash := sha256.Sum256(data)
	switch key := pub.(type) {
	case *ecdsa.PublicKey:
		var s ecdsaSig
		if _, err := asn1.Unmarshal(sig, &s); err != nil {
			return fmt.Errorf("ecdsa sig parse: %w", err)
		}
		if !ecdsa.Verify(key, hash[:], s.R, s.S) {
			return errors.New("ecdsa: invalid signature")
		}
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sig); err != nil {
			return fmt.Errorf("rsa: %w", err)
		}
	default:
		return fmt.Errorf("unsupported key type %T", pub)
	}
	return nil
}

func waChallengeMatch(encoded string, expected []byte) error {
	got, err := waB64Decode(encoded)
	if err != nil {
		return fmt.Errorf("challenge decode: %w", err)
	}
	if string(got) != string(expected) {
		return errors.New("challenge mismatch")
	}
	return nil
}

func waOriginAllowed(origin string, allowed []string) error {
	for _, a := range allowed {
		if origin == a {
			return nil
		}
	}
	return fmt.Errorf("origin %q not allowed", origin)
}

func waB64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}
