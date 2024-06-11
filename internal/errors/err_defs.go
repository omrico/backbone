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
	ProviderNotFoundError       = newError("ERR.02.001", "provider not found", http.StatusBadRequest)
	EncryptStateError           = newError("ERR.02.002", "cannot encrypt state", http.StatusInternalServerError)
	DecryptStateError           = newError("ERR.02.003", "cannot decrypt state", http.StatusInternalServerError)
	ExtractStateFromCookieError = newError("ERR.02.004", "cannot extract state from cookie", http.StatusBadRequest)
	StateMismatchError          = newError("ERR.02.005", "state from cookie not equal to state from query param", http.StatusInternalServerError)
	TokenExchangeError          = newError("ERR.02.006", "failed to exchange token", http.StatusInternalServerError)
	IDTokenMissingError         = newError("ERR.02.007", "failed to extract ID token from response", http.StatusBadRequest)
	VerifyTokenError            = newError("ERR.02.008", "failed to verify ID token", http.StatusInternalServerError)
	ExtractClaimsFromTokenError = newError("ERR.02.009", "failed to extract claims from token", http.StatusInternalServerError)
	TokenSignError              = newError("ERR.02.010", "failed to sign new Backbone ID token", http.StatusInternalServerError)

	// 03 K8s sync
	GetUserRolesError = newError("ERR.04.001", "failed to get mapped user roles", http.StatusInternalServerError)
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
