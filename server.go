package pyritego

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"github.com/mo-crystal/pyrite-go/utils"
)

type ClientData struct {
	Ip           *net.UDPAddr
	LastAccept   int64
	Sequence     int
	SequenceBuff map[int]chan *PrtPackage
}

type Server struct {
	listener    *net.UDPConn
	port        int
	sessionlen  int
	maxLifeTime int64
	occupied    map[string]bool
	router      map[string]func(PrtPackage) *PrtPackage
	timeout     time.Duration

	cdata map[string]*ClientData
}

var (
	ErrServerUDPStartingFailed = errors.New("fail to start udp server")
	ErrServerTellClientTimeout = errors.New("server tell client timeout")
)

//func processAlive(pkage PrtPackage)*PrtPackage

func NewServer(port int, maxTime int64, timeout time.Duration) (*Server, error) {

	var server Server
	server.port = port
	server.router = make(map[string]func(PrtPackage) *PrtPackage)
	server.cdata = make(map[string]*ClientData)
	server.maxLifeTime = maxTime
	server.timeout = timeout
	//server.router["prt-alive"] = processAlive
	return &server, nil
}

func (s *Server) AddRouter(identifier string, controller func(PrtPackage) *PrtPackage) bool {
	if strings.Index(identifier, "prt-") == 0 {
		return false
	}

	s.router[identifier] = controller
	return true
}

func (s *Server) GenerateSession() string {
	var ret string
	for {
		ret = utils.RandomString(s.sessionlen)
		if _, ok := s.occupied[string(ret)]; !ok {
			s.occupied[string(ret)] = true
			break
		}
	}
	return ret
}

func (s *Server) getSequence(session string) int {
	s.cdata[session].Sequence += 1
	return s.cdata[session].Sequence - 1
}

func (s *Server) Tell(session string, identifier, body string) {
	rBytes := PrtPackage{
		Session:    session,
		Identifier: identifier,
		sequence:   -1,
		Body:       body,
	}.ToBytes()
	if len(rBytes) > MAX_TRANSMIT_SIZE {
		panic("package too long")
	}

	s.listener.Write(rBytes)
}

// 向对方发送信息，并且期待 ACK
//
// 此函数会阻塞线程
func (s *Server) Promise(session string, identifier, body string) (string, error) {
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
		return "", ErrContentOverflowed
	}

	s.listener.WriteToUDP(reqBytes, s.cdata[session].Ip)
	s.cdata[session].SequenceBuff[s.cdata[session].Sequence] = make(chan *PrtPackage)
	ch := make(chan bool)
	go Timer(s.timeout, ch, false)
	go func(err *error, ch chan bool) {
		defer func() { recover() }()
		response = <-s.cdata[session].SequenceBuff[s.cdata[session].Sequence]
		ch <- true
	}(&err, ch)
	ok := <-ch
	if !ok {
		return "", ErrServerTellClientTimeout
	}
	return response.Body, nil
}

func (s *Server) processAck(response *PrtPackage) {
	session := response.Session
	ch, ok := s.cdata[session].SequenceBuff[response.sequence]
	if !ok {
		return
	}

	ch <- response
	close(ch)
	delete(s.cdata[session].SequenceBuff, response.sequence)
}

func (s *Server) process(addr *net.UDPAddr, recv []byte) {
	prtPack, err := CastToPrtpackage(recv)
	if err != nil {
		return
	}
	now := time.Now().UnixMicro()
	nowSession := prtPack.Session
	if prtPack.Session == "" {
		nowSession = s.GenerateSession()
		s.cdata[nowSession] = &ClientData{
			Ip:           addr,
			LastAccept:   now,
			Sequence:     0,
			SequenceBuff: make(map[int]chan *PrtPackage),
		}
	}

	if prtPack.Identifier == "prt-ack" {
		s.processAck(prtPack)
		return
	}

	f, ok := s.router[prtPack.Identifier]
	if !ok {
		return
	}

	resp := f(*prtPack)
	if resp == nil {
		return
	}

	resp.Session = nowSession
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
	for k, v := range s.cdata {
		if nowTime-v.LastAccept >= s.maxLifeTime {
			delete(s.cdata, k)
		}
	}
}
