package response

import (
	"mictract/enum"
	"net/http"
	"time"
)

type Response struct {
	Code 	int 		`json:"code"`
	Message string		`json:"message"`
	Payload	interface{}	`json:"payload"`
	Time 	time.Time	`json:"time"`
}

type ResponseWithStatus struct {
	response   Response
	httpStatus int
}

func Ok() *ResponseWithStatus {
	return &ResponseWithStatus{
		httpStatus: http.StatusOK,
		response: Response{
			Code: enum.CodeOk,
			Time: time.Now(),
		},
	}
}

func Err(status int, code int) *ResponseWithStatus {
	msg, ok := enum.CodeMessage[code]
	if !ok {
		msg = "failure"
	}

	return &ResponseWithStatus{
		httpStatus: status,
		response: Response{
			Code: code,
			Message: msg,
			Time: time.Now(),
		},
	}
}

func (r *ResponseWithStatus) SetMessage(message string) *ResponseWithStatus {
	r.response.Message = message
	return r
}

func (r *ResponseWithStatus) SetPayload(payload interface{}) *ResponseWithStatus {
	r.response.Payload = payload
	return r
}

func (r *ResponseWithStatus) Result(f func(int, interface{})) {
	f(r.httpStatus, r.response)
}