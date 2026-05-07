package validator

import (
	"errors"
	"fmt"
	"strings"
)

type FieldError struct {
	Path    string
	Message string
}

func (e FieldError) Error() string {
	if e.Path == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

type FieldErrors []FieldError

func (e FieldErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	parts := make([]string, 0, len(e))
	for _, err := range e {
		parts = append(parts, err.Error())
	}

	return strings.Join(parts, "; ")
}

func (e FieldErrors) Err() error {
	if len(e) == 0 {
		return nil
	}

	return e
}

func RequiredString(path, value string) error {
	if strings.TrimSpace(value) == "" {
		return FieldError{Path: path, Message: "is required"}
	}

	return nil
}

func Append(errs FieldErrors, err error) FieldErrors {
	if err == nil {
		return errs
	}

	var fieldErr FieldError
	if errors.As(err, &fieldErr) {
		return append(errs, fieldErr)
	}

	return append(errs, FieldError{Message: err.Error()})
}
