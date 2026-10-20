package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"

	"github.com/gin-gonic/gin"
)

func TestHandleErrorReturnsSafeInternalErrorWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "request-123")
		c.Next()
	})
	router.GET("/error", func(c *gin.Context) {
		HandleError(c, errors.New("sql password=secret stack trace"))
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/error", nil)
	router.ServeHTTP(record, request)

	if record.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", record.Code)
	}
	body := record.Body.String()
	if body != `{"success":false,"code":"common.internal_server_error","message":"internal server error","request_id":"request-123"}` {
		t.Fatalf("unexpected safe error response: %s", body)
	}
	if containsSensitiveError(body) {
		t.Fatal("internal error details leaked to client")
	}
}

func TestHandleErrorMapsKindToStablePublicContract(t *testing.T) {
	cases := []struct {
		name    string
		kind    faults.Kind
		status  int
		code    string
		message string
	}{
		{name: "bad request", kind: faults.KindInvalidInput, status: http.StatusBadRequest, code: "common.invalid_request", message: "invalid request"},
		{name: "unauthorized", kind: faults.KindUnauthenticated, status: http.StatusUnauthorized, code: "common.unauthorized", message: "unauthorized"},
		{name: "not found", kind: faults.KindNotFound, status: http.StatusNotFound, code: "common.not_found", message: "resource not found"},
		{name: "conflict", kind: faults.KindConflict, status: http.StatusConflict, code: "common.conflict", message: "request conflicts with current state"},
		{name: "unavailable", kind: faults.KindUnavailable, status: http.StatusServiceUnavailable, code: "common.service_unavailable", message: "service temporarily unavailable"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/error", func(c *gin.Context) {
				HandleError(c, faults.New(testCase.kind, "", errors.New("internal detail")))
			})

			record := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/error", nil)
			router.ServeHTTP(record, request)

			expected := `{"success":false,"code":"` + testCase.code + `","message":"` + testCase.message + `","request_id":null}`
			if record.Body.String() != expected {
				t.Fatalf("expected %s, got %s", expected, record.Body.String())
			}
		})
	}
}

func containsSensitiveError(body string) bool {
	return body == "" ||
		contains(body, "sql password") ||
		contains(body, "stack trace")
}

func TestHandleErrorUsesTypedPublicCodeAndSafeMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/error", func(c *gin.Context) {
		err := faults.New(faults.KindConflict, errorconst.AuthEmailAlreadyRegistered, errors.New("duplicate email detail"))
		HandleError(c, err)
	})

	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/error", nil)
	router.ServeHTTP(record, request)

	if record.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", record.Code)
	}
	if record.Body.String() != `{"success":false,"code":"auth.email_already_registered","message":"email is already registered","request_id":null}` {
		t.Fatalf("unexpected typed error response: %s", record.Body.String())
	}
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
