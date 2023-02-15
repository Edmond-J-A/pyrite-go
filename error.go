package pyritego

import "errors"

var (
	ErrContentOverflowed = errors.New("content overflowed")
	ErrTimeout           = errors.New("pyrite counterpart timeouted")
	ErrIllegalOperation  = errors.New("illegal operation")
)
