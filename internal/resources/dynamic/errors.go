package dynamic

import (
	"errors"
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ apierrors.APIStatus = (*APIError)(nil)

// APIError is an error that wraps a Kubernetes API status.
// It formats the error message in a more readable format
// (since Kubernetes natively will only log the message, which could sometimes be simply "unknown").
type APIError struct {
	status metav1.Status
}

// Error implements the error interface.
// It logs the code & reason in addition to the message,
// which could be simply "unknown" in some cases.
func (e APIError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.status.Code, string(e.status.Reason), e.status.Message)
}

// Status implements the apierrors.APIStatus interface.
func (e APIError) Status() metav1.Status {
	return e.status
}

// ParseStatusError parses a Kubernetes API status from an error.
func ParseStatusError(err error) error {
	if err == nil {
		return nil
	}

	if status, ok := err.(apierrors.APIStatus); ok || errors.As(err, &status) {
		return APIError{status.Status()}
	}

	return APIError{
		status: metav1.Status{
			Status:  metav1.StatusFailure,
			Reason:  metav1.StatusReasonUnknown,
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		},
	}
}
