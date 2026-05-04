// nolint
package apperror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("creates service error with all fields", func(t *testing.T) {
		code := 500
		message := "internal server error"
		cause := errors.New("database connection failed")

		err := New(code, message, cause)

		assert.NotNil(t, err)
		assert.Equal(t, code, err.StatusCode())
		assert.Equal(t, message, err.Error())
		assert.Equal(t, fmt.Sprintf("error %d: %s (caused by: %v)", code, message, cause), err.Caused())
	})

	t.Run("creates service error with nil cause", func(t *testing.T) {
		code := 404
		message := "not found"

		err := New(code, message, nil)

		assert.NotNil(t, err)
		assert.Equal(t, code, err.StatusCode())
		assert.Equal(t, message, err.Error())
		assert.Equal(t, fmt.Sprintf("error %d: %s (caused by: unknown)", code, message), err.Caused())
	})

	t.Run("creates service error with empty message", func(t *testing.T) {
		code := 400
		cause := errors.New("validation failed")

		err := New(code, "", cause)

		assert.NotNil(t, err)
		assert.Equal(t, code, err.StatusCode())
		assert.Equal(t, "", err.Error())
		assert.Equal(t, fmt.Sprintf("error %d: %s (caused by: %v)", code, "", cause), err.Caused())
	})

	t.Run("error method returns correct format", func(t *testing.T) {
		err := New(500, "internal error", nil)

		expected := "internal error"
		assert.Equal(t, 500, err.StatusCode())
		assert.Equal(t, expected, err.Error())
	})

	t.Run("caused method returns correct format with cause", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := New(500, "internal error", cause)

		expected := "error 500: internal error (caused by: connection timeout)"
		assert.Equal(t, expected, err.Caused())
	})

	t.Run("unwrap returns cause error", func(t *testing.T) {
		cause := errors.New("original error")
		err := New(500, "internal error", cause)

		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestAppError_CausedBy(t *testing.T) {
	t.Run("creates new service error with updated cause", func(t *testing.T) {
		originalErr := New(500, "internal error", errors.New("original cause"))
		newCause := errors.New("new cause")

		updatedErr := originalErr.CausedBy(newCause)

		assert.NotNil(t, updatedErr)
		assert.Equal(t, 500, updatedErr.StatusCode())
		assert.Equal(t, "internal error", updatedErr.Error())
		assert.Equal(t, newCause, updatedErr.Unwrap())
		assert.Equal(t, "error 500: internal error (caused by: new cause)", updatedErr.Caused())
	})

	t.Run("creates new service error with nil cause", func(t *testing.T) {
		originalErr := New(404, "not found", errors.New("original cause"))

		updatedErr := originalErr.CausedBy(nil)

		assert.NotNil(t, updatedErr)
		assert.Equal(t, 404, updatedErr.StatusCode())
		assert.Equal(t, "not found", updatedErr.Error())
		assert.Nil(t, updatedErr.Unwrap())
		assert.Equal(t, "error 404: not found (caused by: unknown)", updatedErr.Caused())
	})

	t.Run("does not modify original error", func(t *testing.T) {
		originalCause := errors.New("original cause")
		originalErr := New(400, "bad request", originalCause)
		newCause := errors.New("new cause")

		updatedErr := originalErr.CausedBy(newCause)

		assert.Equal(t, originalCause, originalErr.Unwrap())
		assert.Equal(t, newCause, updatedErr.Unwrap())
		assert.NotEqual(t, originalErr, updatedErr)
	})

	t.Run("preserves status code and message", func(t *testing.T) {
		originalErr := New(422, "validation error", nil)
		newCause := errors.New("field validation failed")

		updatedErr := originalErr.CausedBy(newCause)

		assert.Equal(t, originalErr.StatusCode(), updatedErr.StatusCode())
		assert.Equal(t, originalErr.Error(), updatedErr.Error())
		assert.Equal(t, newCause, updatedErr.Unwrap())
	})
}

func TestAppError_SetCaller(t *testing.T) {
	t.Run("sets caller on service error", func(t *testing.T) {
		err := New(500, "internal error", nil).(*AppError)
		caller := "main.go:123"

		err.SetCaller(caller)

		assert.Equal(t, caller, err.Caller)
	})

	t.Run("updates caller when called multiple times", func(t *testing.T) {
		err := New(404, "not found", nil).(*AppError)
		firstCaller := "handler.go:45"
		secondCaller := "service.go:67"

		err.SetCaller(firstCaller)
		assert.Equal(t, firstCaller, err.Caller)

		err.SetCaller(secondCaller)
		assert.Equal(t, secondCaller, err.Caller)
	})

	t.Run("sets empty string as caller", func(t *testing.T) {
		err := New(400, "bad request", nil).(*AppError)

		err.SetCaller("")

		assert.Equal(t, "", err.Caller)
	})

	t.Run("preserves other fields when setting caller", func(t *testing.T) {
		cause := errors.New("original cause")
		err := New(422, "validation error", cause).(*AppError)
		caller := "validator.go:89"

		err.SetCaller(caller)

		assert.Equal(t, 422, err.StatusCode())
		assert.Equal(t, "validation error", err.Error())
		assert.Equal(t, cause, err.Unwrap())
		assert.Equal(t, caller, err.Caller)
	})
}

func TestAppError_CalledAt(t *testing.T) {
	t.Run("returns caller set by New", func(t *testing.T) {
		err := New(500, "internal error", nil)
		caller := err.(*AppError).Caller

		assert.Equal(t, caller, err.CalledAt())
	})

	t.Run("returns updated caller after SetCaller", func(t *testing.T) {
		err := New(404, "not found", nil).(*AppError)
		newCaller := "custom.go:42"
		err.SetCaller(newCaller)

		assert.Equal(t, newCaller, err.CalledAt())
	})

	t.Run("returns empty string if caller is set to empty", func(t *testing.T) {
		err := New(400, "bad request", nil).(*AppError)
		err.SetCaller("")

		assert.Equal(t, "", err.CalledAt())
	})

	t.Run("returns correct caller after CausedBy", func(t *testing.T) {
		originalErr := New(422, "validation error", nil)
		newCause := errors.New("field validation failed")
		updatedErr := originalErr.CausedBy(newCause).(*AppError)

		// The caller should be set by CausedBy (runtime.Caller(1))
		assert.NotEmpty(t, updatedErr.CalledAt())
	})
}

func TestStatusCodeValidation(t *testing.T) {
	t.Run("accepts valid HTTP status codes", func(t *testing.T) {
		validCodes := []int{200, 400, 404, 500, 502}

		for _, code := range validCodes {
			err := New(code, "test message", nil)
			assert.NotNil(t, err)
			assert.Equal(t, code, err.StatusCode())
		}
	})

	t.Run("rejects invalid status codes below 100", func(t *testing.T) {
		invalidCodes := []int{-1, 0, 99}

		for _, code := range invalidCodes {
			err := New(code, "test message", nil)
			assert.NotNil(t, err)
			// Should fallback to 500
			assert.Equal(t, 500, err.StatusCode())
			assert.Contains(t, err.Error(), "invalid status code")
		}
	})

	t.Run("rejects invalid status codes above 599", func(t *testing.T) {
		invalidCodes := []int{600, 1000, 9999}

		for _, code := range invalidCodes {
			err := New(code, "test message", nil)
			assert.NotNil(t, err)
			// Should fallback to 500
			assert.Equal(t, 500, err.StatusCode())
			assert.Contains(t, err.Error(), "invalid status code")
		}
	})
}

func TestMessageValidation(t *testing.T) {
	t.Run("accepts normal length messages", func(t *testing.T) {
		message := "This is a normal error message"
		err := New(400, message, nil)
		assert.Equal(t, message, err.Error())
	})

	t.Run("handles too long messages", func(t *testing.T) {
		// Create a message longer than 1000 characters
		longMessage := ""
		for i := 0; i < 1200; i++ {
			longMessage += "a"
		}

		err := New(400, longMessage, nil)
		assert.Equal(t, "error message validation failed", err.Error())
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewDomainError creates correct error", func(t *testing.T) {
		err := NewDomainError("INVALID_ENTITY", "entity validation failed", nil)
		assert.Equal(t, 422, err.StatusCode())
		assert.Equal(t, "[INVALID_ENTITY] entity validation failed", err.Error())
	})

	t.Run("NewApplicationError creates correct error", func(t *testing.T) {
		err := NewApplicationError("SERVICE_ERROR", "service unavailable", nil)
		assert.Equal(t, 500, err.StatusCode())
		assert.Equal(t, "[SERVICE_ERROR] service unavailable", err.Error())
	})

	t.Run("NewInfrastructureError creates correct error", func(t *testing.T) {
		err := NewInfrastructureError("DB_ERROR", "database connection failed", nil)
		assert.Equal(t, 502, err.StatusCode())
		assert.Equal(t, "[DB_ERROR] database connection failed", err.Error())
	})

	t.Run("NewValidationError creates correct error", func(t *testing.T) {
		err := NewValidationError("REQUIRED_FIELD", "field is required", nil)
		assert.Equal(t, 422, err.StatusCode())
		assert.Equal(t, "[REQUIRED_FIELD] field is required", err.Error())
	})

	t.Run("NewInternalError creates correct error", func(t *testing.T) {
		err := NewInternalError("INT", "internal", nil)
		assert.Equal(t, 502, err.StatusCode())
		assert.Equal(t, "[INT] internal", err.Error())
	})

	t.Run("NewNotFoundError creates correct error", func(t *testing.T) {
		err := NewNotFoundError("User", nil)
		assert.Equal(t, 404, err.StatusCode())
		assert.Equal(t, "User not found", err.Error())
	})

	t.Run("NewUnauthorizedError creates correct error", func(t *testing.T) {
		err := NewUnauthorizedError("Invalid credentials", nil)
		assert.Equal(t, 401, err.StatusCode())
		assert.Equal(t, "Invalid credentials", err.Error())
	})

	t.Run("NewForbiddenError creates correct error", func(t *testing.T) {
		err := NewForbiddenError("Access denied", nil)
		assert.Equal(t, 403, err.StatusCode())
		assert.Equal(t, "Access denied", err.Error())
	})
}

func TestCausedByWithNilError(t *testing.T) {
	t.Run("CausedBy with nil error creates new error with nil cause", func(t *testing.T) {
		originalErr := New(400, "original error", nil)
		result := originalErr.CausedBy(nil)

		// Should create new error with nil cause
		assert.Equal(t, originalErr.StatusCode(), result.StatusCode())
		assert.Equal(t, originalErr.Error(), result.Error())
		assert.Nil(t, result.Unwrap())
		assert.Contains(t, result.Caused(), "unknow") // Caused() still shows "unknow" for display
	})
}
