package pyritego

import (
	"net"
)

type Server struct {
	listener    net.UDPConn
	router      map[string]func(Request) Response
	session     map[string]interface{}
	rtt         map[string]int64
	lastAccept  map[string]int64
	maxLifeTime int64
}

func NewServer(port int, enableSession bool, maxTime int64) (*Server, error) {

	var server Server
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port})
	server.listener = listener
	if err != nil {
		return nil, ErrUDPServerStartingFailed
	}

	if enableSession {
		server.session = make(map[string]interface{})
	}
	server.maxLifeTime = maxTime

	return &server, nil
}

func (s *Server) AddRouter(identifier string, controller func(Request) Response)
func (s *Server) SetSession(session string, data interface{})
func (s *Server) DelSession(session string)
func (s *Server) Tell(remote *net.UDPAddr, identifier, body string) (Response, error)
func (s *Server) Start()
func (s *Server) GC()
