package comments

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type publicHandler struct {
	svc *Service
}

func newPublicHandler(svc *Service) *publicHandler {
	return &publicHandler{svc: svc}
}

func (h *publicHandler) ListComments(c *gin.Context) {
	slug := c.Query("post_slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "post_slug is required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	comments, total, err := h.svc.store.ListComments(c.Request.Context(), slug, StatusApproved, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
	})
}

func (h *publicHandler) CreateComment(c *gin.Context) {
	var req struct {
		PostSlug string `json:"post_slug" binding:"required"`
		Author   string `json:"author" binding:"required"`
		Content  string `json:"content" binding:"required"`
		ReplyTo  string `json:"reply_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: post_slug, author, content are required"})
		return
	}

	req.Author = strings.TrimSpace(req.Author)
	req.Content = strings.TrimSpace(req.Content)
	if len(req.Author) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author too long (max 50)"})
		return
	}
	if len(req.Content) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content too long (max 2000)"})
		return
	}

	if req.ReplyTo != "" {
		parent, err := h.svc.store.GetComment(c.Request.Context(), req.ReplyTo)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "reply target not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if parent.PostSlug != req.PostSlug {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reply target belongs to a different post"})
			return
		}
	}

	ip := c.ClientIP()
	ipHash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))

	comment := &Comment{
		ID:       uuid.NewString(),
		PostSlug: req.PostSlug,
		Author:   req.Author,
		Content:  req.Content,
		ReplyTo:  req.ReplyTo,
		IPHash:   ipHash,
		Status:   StatusApproved,
	}
	if err := h.svc.store.InsertComment(c.Request.Context(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

type adminHandler struct {
	svc *Service
}

func newAdminHandler(svc *Service) *adminHandler {
	return &adminHandler{svc: svc}
}

func (h *adminHandler) ListAll(c *gin.Context) {
	slug := c.Query("post_slug")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	comments, total, err := h.svc.store.ListComments(c.Request.Context(), slug, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
	})
}

func (h *adminHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	switch req.Status {
	case StatusApproved, StatusPending, StatusSpam, StatusDeleted:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	if err := h.svc.store.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *adminHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.store.DeleteComment(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
