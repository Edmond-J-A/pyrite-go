package pyritego

import "net"

type Request struct {
	Owner       *Server
	Remote      *net.UDPAddr
	Session     string
	SessionData interface{}
	Body        string
}

func (r *Request) SetSession(data interface{}) {

}

func (r *Request) DelSession() {

}
