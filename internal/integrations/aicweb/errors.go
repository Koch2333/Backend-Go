package aicweb

import "errors"

// 错误码与文案对齐 aicweb（示例）
const (
	ErrCodeSuccess  = 0
	ErrCodeBadReq   = 400
	ErrCodeUnauthed = 401
	ErrCodeNotFound = 404
	ErrCodeInternal = 500

	// 业务自定义示例
	ErrCodeEmailUsed = 1001
	ErrCodeUnknown   = 500 // 兜底
)

var (
	ErrSuccess             = newError(ErrCodeSuccess, "ok")
	ErrBadRequest          = newError(ErrCodeBadReq, "Bad Request")
	ErrUnauthorized        = newError(ErrCodeUnauthed, "Unauthorized")
	ErrNotFound            = newError(ErrCodeNotFound, "Not Found")
	ErrInternalServerError = newError(ErrCodeInternal, "Internal Server Error")
	ErrEmailAlreadyUse     = newError(ErrCodeEmailUsed, "The email is already in use.")
)

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string { return e.Message }

var errorCodeMap = map[error]int{}

func newError(code int, msg string) error {
	err := errors.New(msg)
	errorCodeMap[err] = code
	return err
}
