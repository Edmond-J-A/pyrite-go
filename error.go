package pyritego

import "errors"

var (
	ErrUDPClientBindingFailed = errors.New("cannot bind udp")
	ErrInvalidPyriteServer    = errors.New("invalid pyrite server")
)
