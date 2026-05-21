package application

import "fmt"

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func NewNotFoundError(msg string) *AppError {
	return &AppError{Code: 404, Message: msg}
}

func NewUnauthorizedError(msg string) *AppError {
	return &AppError{Code: 401, Message: msg}
}

func NewForbiddenError(msg string) *AppError {
	return &AppError{Code: 403, Message: msg}
}

func NewBadRequestError(msg string) *AppError {
	return &AppError{Code: 400, Message: msg}
}

func NewInternalError(msg string, err error) *AppError {
	return &AppError{Code: 500, Message: msg, Err: err}
}
