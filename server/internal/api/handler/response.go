package handler

// Response is the unified API response envelope.
// code=0 means success; any other value indicates an error.
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// OK wraps data in a success response (code=0).
func OK(data interface{}) Response {
	return Response{Code: 0, Msg: "ok", Data: data}
}

// Fail builds an error response with the given code and message.
func Fail(code int, msg string) Response {
	return Response{Code: code, Msg: msg, Data: nil}
}
