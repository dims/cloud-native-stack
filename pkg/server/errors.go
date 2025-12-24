package server

import (
	"net/http"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
	"github.com/google/uuid"
)

// Error codes as constants
const (
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeInvalidRequest     = "INVALID_REQUEST"
	ErrCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
)

// ErrorResponse represents error responses as per OpenAPI spec
type ErrorResponse struct {
	Code      string                 `json:"code" yaml:"code"`
	Message   string                 `json:"message" yaml:"message"`
	Details   map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
	RequestID string                 `json:"requestId" yaml:"requestId"`
	Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
	Retryable bool                   `json:"retryable" yaml:"retryable"`
}

// writeError writes error response
func WriteError(w http.ResponseWriter, r *http.Request, statusCode int,
	code, message string, retryable bool, details map[string]interface{}) {

	requestID, _ := r.Context().Value(contextKeyRequestID).(string)
	if requestID == "" {
		requestID = uuid.New().String()
	}

	errResp := ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
		Retryable: retryable,
	}

	serializer.RespondJSON(w, statusCode, errResp)
}
