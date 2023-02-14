package pyritego

import "errors"

var (
	ErrServerUDPStartingFailed = errors.New("fail to start udp server")
	ErrUDPClientBindingFailed  = errors.New("cannot bind udp")
	ErrInvalidPyriteServer     = errors.New("invalid pyrite server")
)
