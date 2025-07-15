package erro_test

import (
	"net/http"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestHTTPCode(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, http.StatusOK},
		{"validation error", erro.New("test", erro.ClassValidation), http.StatusBadRequest},
		{"not found error", erro.New("test", erro.ClassNotFound), http.StatusNotFound},
		{"already exists error", erro.New("test", erro.ClassAlreadyExists), http.StatusConflict},
		{"permission denied error", erro.New("test", erro.ClassPermissionDenied), http.StatusForbidden},
		{"unauthenticated error", erro.New("test", erro.ClassUnauthenticated), http.StatusUnauthorized},
		{"timeout error", erro.New("test", erro.ClassTimeout), http.StatusGatewayTimeout},
		{"conflict error", erro.New("test", erro.ClassConflict), http.StatusConflict},
		{"rate limited error", erro.New("test", erro.ClassRateLimited), http.StatusTooManyRequests},
		{"temporary error", erro.New("test", erro.ClassTemporary), http.StatusServiceUnavailable},
		{"unavailable error", erro.New("test", erro.ClassUnavailable), http.StatusServiceUnavailable},
		{"internal error", erro.New("test", erro.ClassInternal), http.StatusInternalServerError},
		{"cancelled error", erro.New("test", erro.ClassCancelled), 499},
		{"not implemented error", erro.New("test", erro.ClassNotImplemented), http.StatusNotImplemented},
		{"security error", erro.New("test", erro.ClassSecurity), http.StatusForbidden},
		{"critical error", erro.New("test", erro.ClassCritical), http.StatusInternalServerError},
		{"external error", erro.New("test", erro.ClassExternal), http.StatusBadGateway},
		{"data loss error", erro.New("test", erro.ClassDataLoss), http.StatusInternalServerError},
		{"resource exhausted error", erro.New("test", erro.ClassResourceExhausted), http.StatusTooManyRequests},
		{"user input category", erro.New("test", erro.CategoryUserInput), http.StatusBadRequest},
		{"auth category", erro.New("test", erro.CategoryAuth), http.StatusUnauthorized},
		{"database category", erro.New("test", erro.CategoryDatabase), http.StatusInternalServerError},
		{"network category", erro.New("test", erro.CategoryNetwork), http.StatusBadGateway},
		{"api category", erro.New("test", erro.CategoryAPI), http.StatusBadGateway},
		{"business logic category", erro.New("test", erro.CategoryBusinessLogic), http.StatusUnprocessableEntity},
		{"cache category", erro.New("test", erro.CategoryCache), http.StatusServiceUnavailable},
		{"config category", erro.New("test", erro.CategoryConfig), http.StatusInternalServerError},
		{"external category", erro.New("test", erro.CategoryExternal), http.StatusBadGateway},
		{"security category", erro.New("test", erro.CategorySecurity), http.StatusForbidden},
		{"payment category", erro.New("test", erro.CategoryPayment), http.StatusPaymentRequired},
		{"storage category", erro.New("test", erro.CategoryStorage), http.StatusInsufficientStorage},
		{"processing category", erro.New("test", erro.CategoryProcessing), http.StatusUnprocessableEntity},
		{"analytics category", erro.New("test", erro.CategoryAnalytics), http.StatusInternalServerError},
		{"ai category", erro.New("test", erro.CategoryAI), http.StatusInternalServerError},
		{"monitoring category", erro.New("test", erro.CategoryMonitoring), http.StatusInternalServerError},
		{"notifications category", erro.New("test", erro.CategoryNotifications), http.StatusInternalServerError},
		{"events category", erro.New("test", erro.CategoryEvents), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := erro.HTTPCode(tc.err)
			if code != tc.expected {
				t.Errorf("Expected HTTP code %d, got %d", tc.expected, code)
			}
		})
	}
}
