package apperror

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// Common HTTP status codes for application errors
const (
	// 4xx Client Error
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusConflict            = 409
	StatusUnprocessableEntity = 422
	StatusTooManyRequests     = 429

	// 5xx Server Error
	StatusInternalServerError = 500
	StatusNotImplemented      = 501
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
	StatusGatewayTimeout      = 504
)

// Error types for categorization
const (
	DomainError         = "DOMAIN_ERROR"
	ApplicationError    = "APPLICATION_ERROR"
	InfrastructureError = "INFRASTRUCTURE_ERROR"
	ValidationError     = "VALIDATION_ERROR"
)

// Error message format for categorized errors
const (
	ErrorMessageFormat = "[%s] %s"
)

// isValidHTTPStatusCode validates if the status code is a valid HTTP status code (100-599)
func isValidHTTPStatusCode(code int) bool {
	return code >= 100 && code <= 599
}

// validateStatusCode validates the status code and returns an error if invalid
func validateStatusCode(code int) error {
	if !isValidHTTPStatusCode(code) {
		return fmt.Errorf("invalid status code: %d (must be between 100 and 599)", code)
	}
	return nil
}

// validateMessage validates the error message
func validateMessage(message string) error {
	if len(message) > 1000 {
		return errors.New("error message too long (max 1000 characters)")
	}
	return nil
}

type IAppError interface {
	StatusCode() int
	Error() string
	Unwrap() error
	CausedBy(err error) IAppError
	Caused() string
	ErrCause() error
	SetCaller(string)
	CalledAt() string
}

// AppError represents a custom error with additional context
type AppError struct {
	Code    int
	Message string
	Cause   error
	Caller  string
}

// New creates a new AppError with validation
// statuscode: HTTP status code (100-599)
// public message: user-friendly error message
// causedBy: underlying error that caused this error
func New(code int, message string, causedBy error) IAppError {
	// Validate status code
	if err := validateStatusCode(code); err != nil {
		// If status code is invalid, use 500 as fallback
		code = StatusInternalServerError
		message = fmt.Sprintf("invalid status code provided: %v", err)
		causedBy = err
	}

	// Validate message
	if err := validateMessage(message); err != nil {
		message = "error message validation failed"
		if causedBy == nil {
			causedBy = err
		}
	}

	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", filepath.Base(file), line)

	return &AppError{
		Code:    code,
		Message: message,
		Cause:   causedBy,
		Caller:  caller,
	}
}

// Error implements the error interface
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause error
func (e *AppError) Unwrap() error {
	return e.Cause
}

func (e *AppError) StatusCode() int {
	return e.Code
}

func (e *AppError) SetCaller(val string) {
	e.Caller = val
}

func (e *AppError) CalledAt() string {
	return e.Caller
}

func (e *AppError) Caused() string {
	cause := e.Cause
	if cause == nil {
		cause = errors.New("unknown")
	}
	return fmt.Sprintf("error %d: %s (caused by: %v)", e.Code, e.Message, cause)
}

func (e *AppError) ErrCause() error {
	return errors.New(e.Caused())
}

func (e *AppError) CausedBy(err error) IAppError {
	newError := New(e.Code, e.Message, err).(*AppError)

	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", filepath.Base(file), line)
	newError.SetCaller(caller)
	return newError
}

// Helper functions for common error types

// NewDomainError creates a domain error (business logic errors)
func NewDomainError(code string, message string, causedBy error) IAppError {
	return New(StatusUnprocessableEntity, fmt.Sprintf(ErrorMessageFormat, code, message), causedBy)
}

// NewApplicationError creates an application error (application logic errors)
func NewApplicationError(code string, message string, causedBy error) IAppError {
	return New(StatusInternalServerError, fmt.Sprintf(ErrorMessageFormat, code, message), causedBy)
}

// NewInfrastructureError creates an infrastructure error (external service/database errors)
func NewInfrastructureError(code string, message string, causedBy error) IAppError {
	return New(StatusBadGateway, fmt.Sprintf(ErrorMessageFormat, code, message), causedBy)
}

// NewValidationError creates a validation error (input validation errors)
func NewValidationError(code string, message string, causedBy error) IAppError {
	return New(StatusUnprocessableEntity, fmt.Sprintf(ErrorMessageFormat, code, message), causedBy)
}

// NewInternalError creates an internal error (alias for NewInfrastructureError)
func NewInternalError(code string, message string, causedBy error) IAppError {
	return NewInfrastructureError(code, message, causedBy)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string, causedBy error) IAppError {
	return New(StatusNotFound, fmt.Sprintf("%s not found", resource), causedBy)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string, causedBy error) IAppError {
	return New(StatusUnauthorized, message, causedBy)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string, causedBy error) IAppError {
	return New(StatusForbidden, message, causedBy)
}
