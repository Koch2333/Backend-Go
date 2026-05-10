package authflow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	totpDigits = 6
	totpPeriod = 30
)

func genTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// TOTPProvisioningURI returns the otpauth:// URI for QR code display.
func TOTPProvisioningURI(issuer, username, secret string) string {
	label := url.PathEscape(issuer + ":" + username)
	q := url.Values{
		"secret":    {secret},
		"issuer":    {issuer},
		"algorithm": {"SHA1"},
		"digits":    {fmt.Sprint(totpDigits)},
		"period":    {fmt.Sprint(totpPeriod)},
	}
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// VerifyTOTP checks a 6-digit code with ±1 period tolerance.
func VerifyTOTP(secret, code string) bool {
	t := time.Now().Unix() / int64(totpPeriod)
	code = strings.TrimSpace(code)
	for delta := int64(-1); delta <= 1; delta++ {
		if totpCode(secret, uint64(t+delta)) == code {
			return true
		}
	}
	return false
}

func totpCode(secret string, counter uint64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	h := mac.Sum(nil)
	offset := h[len(h)-1] & 0x0f
	bin := int(h[offset]&0x7f)<<24 | int(h[offset+1])<<16 | int(h[offset+2])<<8 | int(h[offset+3])
	return fmt.Sprintf("%0*d", totpDigits, bin%int(math.Pow10(totpDigits)))
}
