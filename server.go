package pyritego

import (
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"net"
	"strings"
	"time"
)

type ClientObject struct {
}

type Server struct {
	listener    *net.UDPConn
	port        int
	sessionlen  int
	maxLifeTime int64
	occupied    map[string]bool
	router      map[string]func(PrtPackage) *PrtPackage

	ip           map[string]*net.UDPAddr
	session      map[string]interface{}
	rtt          map[string]int64
	lastAccept   map[string]int64
	status       map[string]int
	sequence     map[string]int
	sequenceBuff map[string](map[int]chan *PrtPackage)
	timeout      map[string]time.Duration
}

var (
	ErrServerUDPStartingFailed = errors.New("fail to start udp server")
	ErrerverTellSClientTimeout = errors.New("server tell client timeout")
)

func NewServer(port int, maxTime int64) (*Server, error) {

	var server Server
	server.port = port
	server.router = make(map[string]func(PrtPackage) *PrtPackage)

	server.maxLifeTime = maxTime
	return &server, nil
}

func (s *Server) AddRouter(identifier string, controller func(PrtPackage) *PrtPackage) bool {
	if strings.Index(identifier, "prt-") == 0 {
		return false
	}

	s.router[identifier] = controller
	return true
}

func (s *Server) SetSession(session string, data interface{}) {
	s.session[session] = data
}

func (s *Server) DelSession(session string) {
	delete(s.session, session)
}

func (s *Server) GenerateSession() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	ret := make([]rune, s.sessionlen)
	var randInt *big.Int
	for {
		for i := range ret {
			randInt, _ = rand.Int(rand.Reader, big.NewInt(26+26+10))
			ret[i] = letters[randInt.Int64()]
		}
		if _, ok := s.occupied[string(ret)]; !ok { //TODO:进位算法替换
			s.occupied[string(ret)] = true
			break
		}
	}
	return string(ret)
}

func (s *Server) getSequence(session string) int {
	s.sequence[session] += 1
	return s.sequence[session] - 1
}

func (s *Server) Tell(session string, identifier, body string) (*PrtPackage, error) {
	var response *PrtPackage
	var err error
	req := PrtPackage{
		Session:    session,
		Identifier: identifier,
		sequence:   s.getSequence(session),
		Body:       body,
	}

	reqBytes := req.ToBytes()
	if len(reqBytes) > MAX_TRANSMIT_SIZE {
		return nil, ErrContentOverflowed
	}

	s.listener.WriteToUDP(reqBytes, s.ip[session])
	s.sequenceBuff[session][s.sequence[session]] = make(chan *PrtPackage)
	ch := make(chan bool)
	go Timer(s.timeout[session], ch, false)
	go func(err *error, ch chan bool) {
		defer func() { recover() }()
		response = <-s.sequenceBuff[session][s.sequence[session]]
		ch <- true
	}(&err, ch)
	ok := <-ch
	if !ok {
		return nil, ErrerverTellSClientTimeout
	}
	return response, nil
}

func (s *Server) processHello(addr *net.UDPAddr, requst *PrtPackage) error {
	newSession := s.GenerateSession()
	s.status[newSession] = CLIENT_CREATED
	start := time.Now().UnixMicro()

	var response *PrtPackage
	var err error
	s.ip[newSession] = addr
	if response, err = s.Tell(newSession, "prt-hello", string(s.maxLifeTime)); err != nil {
		delete(s.ip, newSession)
		return err
	}

	if response.Identifier != "prt-established" {
		return ErrServerProcotol
	}

	s.rtt[newSession] = time.Now().UnixMicro() - start
	s.status[newSession] = CLIENT_ESTABLISHED
	return nil
}

func (s *Server) processAck(response *PrtPackage) {
	session := response.Session
	ch, ok := s.sequenceBuff[session][response.sequence]
	if !ok {
		return
	}

	ch <- response
	close(ch)
	delete(s.sequenceBuff[session], response.sequence)
}

func (s *Server) process(addr *net.UDPAddr, recv []byte) {

	request, err := CastToPrtpackage(recv)
	if err != nil {
		return
	}

	if request.Identifier == "prt-hello" {
		s.processHello(addr, request)
		return
	}

	if s.status[request.Session] != CLIENT_ESTABLISHED {
		panic("invalid client status")
	}

	response, err := CastToPrtpackage(recv)
	if err != nil {
		return
	}

	if response.Identifier == "prt-ack" {
		s.processAck(response)
		return
	}

	f, ok := s.router[request.Identifier]
	if !ok {
		return
	}

	resp := f(*request)
	if resp == nil {
		return
	}

	resp.Identifier = "prt-ack"
	s.listener.Write(resp.ToBytes())
}

func (s *Server) Start() error {
	listener, err1 := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: s.port})
	if err1 != nil {
		return ErrServerUDPStartingFailed
	}
	s.listener = listener
	recvBuf := make([]byte, MAX_TRANSMIT_SIZE)
	var n int
	var err error
	var addr *net.UDPAddr
	for {
		n, addr, err = listener.ReadFromUDP(recvBuf)
		if err != nil {
			log.Default()
		}
		go s.process(addr, recvBuf[:n])
	}
}

func (s *Server) GC() {
	nowTime := time.Now().UnixMicro()
	for k, v := range s.lastAccept {
		if nowTime-v >= s.maxLifeTime {
			delete(s.lastAccept, k)
			delete(s.rtt, k)
			delete(s.session, k)
			delete(s.status, k)
			delete(s.ip, k)
			delete(s.sequence, k)
			delete(s.timeout, k)
			delete(s.sequenceBuff, k)
			delete(s.occupied, k)
		}
	}
}
