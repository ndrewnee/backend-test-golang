package users

import "errors"

var (
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrUserNotFound      = errors.New("user not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type invalidAmountError struct {
	err error
}

func newInvalidAmountError(err error) error {
	return invalidAmountError{err: err}
}

func (e invalidAmountError) Error() string {
	return e.err.Error()
}

func (e invalidAmountError) Is(target error) bool {
	return target == ErrInvalidAmount
}
