package aicweb

import "time"

// —— 对外响应契约 ——
// 与 aicweb 保持一致：统一为 {code, message, data}
// code: 0 表示成功，非 0 表示失败；message 为文案；data 为对象或空 map

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 成功响应构造
func NewOK(data interface{}) Response {
	if data == nil {
		data = map[string]any{}
	}
	return Response{Code: ErrCodeSuccess, Message: ErrSuccess.Error(), Data: data}
}

// 失败响应构造（若 err 未注册到映射，则降级为 Unknown）
func NewFail(err error, data interface{}) Response {
	if data == nil {
		data = map[string]any{}
	}
	code, ok := errorCodeMap[err]
	if !ok {
		return Response{Code: ErrCodeUnknown, Message: "unknown error", Data: data}
	}
	return Response{Code: code, Message: err.Error(), Data: data}
}

// —— 登录/注册 DTO ——

// dto.go
type LoginRequest struct {
	Username string `json:"username"`        // 兼容 aicweb：邮箱或用户名都填到这里
	Email    string `json:"email,omitempty"` // 兼容你现在的前端
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username"`        // 兼容 aicweb：邮箱或用户名都填到这里
	Email    string `json:"email,omitempty"` // 兼容你现在的前端
	Password string `json:"password" binding:"required"`
}

type LoginResponseData struct {
	AccessToken string `json:"accessToken"`
}

// —— 演示用的用户结构（仅内存）——

type user struct {
	ID        string
	Username  string
	Email     string
	Password  string // 明文存储仅限开发演示；正式环境请改为 hash
	CreatedAt time.Time
}
