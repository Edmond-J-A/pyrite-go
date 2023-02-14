package pyritego

import "net"

type Request struct {
	Owner       *Server
	Remote      *net.UDPAddr
	Session     string
	SessionData interface{}
	Body        string
}
