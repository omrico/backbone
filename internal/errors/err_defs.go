package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorWrapper struct {
	ErrCode    string
	ErrMessage string
	HttpCode   int
}

var (
	// 01 sessions
	UserNotLoggedInError          = newError("ERR.01.001", "user not logged in", http.StatusForbidden)
	CannotExtractSessionInfoError = newError("ERR.01.002", "cannot extract info from session", http.StatusInternalServerError)
	SessionExpiredError           = newError("ERR.01.003", "session expired", http.StatusUnauthorized)

	// 02 oauth...
)

func MakeError(c *gin.Context, err ErrorWrapper) {
	requestID := c.GetString("request-id")
	c.JSON(err.HttpCode, buildErrorResponse(err, requestID))
}

func buildErrorResponse(err ErrorWrapper, requestID string) map[string]string {
	var res = map[string]string{}
	res["errCode"] = err.ErrCode
	res["errMessage"] = err.ErrMessage
	res["requestID"] = requestID
	return res
}

func newError(errCode string, errMessage string, httpCode int) ErrorWrapper {
	return ErrorWrapper{
		ErrCode:    errCode,
		ErrMessage: errMessage,
		HttpCode:   httpCode,
	}
}
