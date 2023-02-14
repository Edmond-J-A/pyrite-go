package pyritego

import "net"

type Server struct {
	ip          net.IP
	router      map[string]func(Request) Response
	session     map[string]interface{}
	rtt         map[string]int64
	lastAccept  map[string]int64
	maxLifeTime int64
}

func NewServer(serverAddr string, enableSession bool, maxTime int64) (Server, error)
func (s *Server) AddRouter(identifier string, controller func(Request) Response)
func (s *Server) SetSession(session string, data interface{})
func (s *Server) DelSession(session string)
func (s *Server) Tell(remote *net.UDPAddr, identifier, body string) (Response, error)
func (s *Server) Start()
func (s *Server) GC()
