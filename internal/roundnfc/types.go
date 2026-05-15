package roundnfc

import "time"

type Badge struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Series      string    `json:"series,omitempty"`
	Type        string    `json:"type,omitempty"`
	StyleKey    string    `json:"styleKey,omitempty"`
	ImageURL    string    `json:"imageUrl,omitempty"`
	Description string    `json:"description,omitempty"`
	SerialNo    string    `json:"serialNo,omitempty"`
	ReleasedAt  string    `json:"releasedAt,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
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
