package errs

import "errors"

// BizError is a business error with a numeric code suitable for JsonResult.
type BizError struct {
	Code  int
	Msg   string
	Cause error
}

func (e *BizError) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func (e *BizError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(code int, msg string) *BizError {
	return &BizError{Code: code, Msg: msg}
}

func Wrap(code int, msg string, cause error) *BizError {
	return &BizError{Code: code, Msg: msg, Cause: cause}
}

func AsBiz(err error) (*BizError, bool) {
	var b *BizError
	if errors.As(err, &b) {
		return b, true
	}
	return nil, false
}
