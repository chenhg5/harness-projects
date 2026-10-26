package apperror

import (
	"errors"
	"fmt"
	"os"
)

type conflictError struct {
	message string
}

func (err *conflictError) Error() string { return err.message }

func (err *conflictError) Is(target error) bool {
	return errors.Is(target, os.ErrExist)
}

func Conflict(format string, values ...any) error {
	return &conflictError{message: fmt.Sprintf(format, values...)}
}
