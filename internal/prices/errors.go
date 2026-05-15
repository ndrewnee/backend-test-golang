package prices

import "errors"

var (
	ErrInvalidAppID        = errors.New("app_id must be positive")
	ErrUnsupportedCurrency = errors.New("unsupported currency")
)
