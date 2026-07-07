package roundnfc

import (
	"encoding/json"
	"time"
)

type Badge struct {
	ID                    string             `json:"id"`
	Title                 string             `json:"title"`
	Series                string             `json:"series,omitempty"`
	Type                  string             `json:"type,omitempty"`
	StyleKey              string             `json:"styleKey,omitempty"`
	ImageURL              string             `json:"imageUrl,omitempty"`
	StyleImageURL         string             `json:"styleImageUrl,omitempty"`
	StyleImageOriginalURL string             `json:"styleImageOriginalUrl,omitempty"`
	Description           string             `json:"description,omitempty"`
	SerialNo              string             `json:"serialNo,omitempty"`
	ReleasedAt            string             `json:"releasedAt,omitempty"`
	CoserBinding          *BadgeCoserBinding `json:"coserBinding,omitempty"`
	CreatedAt             time.Time          `json:"createdAt,omitempty"`
	UpdatedAt             time.Time          `json:"updatedAt,omitempty"`
}

type BadgeStyleTemplate struct {
	Key              string          `json:"key"`
	Label            string          `json:"label"`
	Description      string          `json:"description,omitempty"`
	ImageURL         string          `json:"imageUrl,omitempty"`
	ImageOriginalURL string          `json:"imageOriginalUrl,omitempty"`
	ImagePreviewURL  string          `json:"imagePreviewUrl,omitempty"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	Enabled          bool            `json:"enabled"`
	CreatedAt        time.Time       `json:"createdAt,omitempty"`
	UpdatedAt        time.Time       `json:"updatedAt,omitempty"`
}

type BadgeCoserBinding struct {
	BadgeID        string    `json:"badgeId"`
	CN             string    `json:"cn"`
	PhotoObjectKey string    `json:"photoObjectKey"`
	DeviceID       string    `json:"deviceId,omitempty"`
	TagUID         string    `json:"tagUid,omitempty"`
	WrittenAt      time.Time `json:"writtenAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

const (
	StatusNew      = "new"
	StatusHandled  = "handled"
	StatusRejected = "rejected"
)

type PhotoRequest struct {
	ID             string    `json:"id"`
	BadgeID        string    `json:"badgeId"`
	Name           string    `json:"name"`
	Contact        string    `json:"contact"`
	Message        string    `json:"message,omitempty"`
	Status         string    `json:"status"`
	AttachmentKeys []string  `json:"attachmentKeys,omitempty"`
	IPHash         string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AutographRequest struct {
	ID             string    `json:"id"`
	BadgeID        string    `json:"badgeId"`
	Name           string    `json:"name"`
	Contact        string    `json:"contact"`
	Target         string    `json:"target"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	AttachmentKeys []string  `json:"attachmentKeys,omitempty"`
	IPHash         string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type NFCWrite struct {
	ID             string    `json:"id"`
	BadgeID        string    `json:"badgeId"`
	TagUID         string    `json:"tagUid"`
	NDEFURL        string    `json:"ndefUrl"`
	DeviceID       string    `json:"deviceId"`
	WriteStatus    string    `json:"writeStatus"`
	PhotoObjectKey string    `json:"photoObjectKey,omitempty"`
	WrittenAt      time.Time `json:"writtenAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

type AppToken struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"tokenPrefix"`
	Enabled     bool       `json:"enabled"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type AppPairingConfig struct {
	Protocol    string            `json:"protocol"`
	Version     int               `json:"version"`
	Name        string            `json:"name"`
	ApiBase     string            `json:"apiBase"`
	ApiPrefix   string            `json:"apiPrefix"`
	TokenHeader string            `json:"tokenHeader"`
	Token       string            `json:"token"`
	Endpoints   map[string]string `json:"endpoints"`
	CreatedAt   time.Time         `json:"createdAt"`
}
