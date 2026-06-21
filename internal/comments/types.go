package comments

import "time"

type Comment struct {
	ID        string    `json:"id"`
	PostSlug  string    `json:"post_slug"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	ReplyTo   string    `json:"reply_to,omitempty"`
	IPHash    string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	StatusApproved = "approved"
	StatusPending  = "pending"
	StatusSpam     = "spam"
	StatusDeleted  = "deleted"
)
