package pyritego

import (
	"net"
	"time"
)

type Server struct {
	port        int
	router      map[string]func(Request) Response
	session     map[string]interface{}
	rtt         map[string]int64
	lastAccept  map[string]int64
	maxLifeTime int64
}

func NewServer(port int, maxTime int64) (*Server, error) {

	var server Server
	server.port = port
	server.router = make(map[string]func(Request) Response)

	server.maxLifeTime = maxTime
	return &server, nil
}

func (s *Server) AddRouter(identifier string, controller func(Request) Response) {
	s.router[identifier] = controller
}

func (s *Server) SetSession(session string, data interface{}) {
	s.session[session] = data
}

func (s *Server) DelSession(session string) {
	delete(s.session, session)
}

func (s *Server) Tell(remote *net.UDPAddr, identifier, body string) (Response, error)
func (s *Server) Start() error {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: s.port})
	if err != nil {
		return ErrServerUDPStartingFailed
	}

	return nil
}

func (s *Server) GC() {
	nowTime := time.Now().Unix()
	for k, v := range s.lastAccept {
		if nowTime-v >= s.maxLifeTime {
			delete(s.lastAccept, k)
			delete(s.rtt, k)
			delete(s.session, k)
		}
	}
}
