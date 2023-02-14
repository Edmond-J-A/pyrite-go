package pyritego

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	CLIENT_CREATED     = 0
	CLIENT_ESTABLISHED = 1
)

type Client struct {
	server     net.UDPAddr
	router     map[string]func(Request) *Response
	connection *net.UDPConn
	status     int

	Session     string
	MaxLifeTime int64
	RTT         int64
	timeout     time.Duration
}

var (
	ErrClientIllegalOperation  = errors.New("illegal operation")
	ErrClientUDPBindingFailed  = errors.New("udp binding failed")
	ErrContentOverflowed       = errors.New("content overflowed")
	ErrServerProcotol          = errors.New("invalid server protocol")
	ErrClientTellServerTimeout = errors.New("client tell server timeout")
)

func NewClient(serverAddr net.UDPAddr, timeout time.Duration) (*Client, error) {
	src := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	connection, err := net.DialUDP("udp", src, &serverAddr)
	if err != nil {
		return nil, ErrClientUDPBindingFailed
	}

	router := make(map[string]func(Request) *Response)
	ret := &Client{
		server:     serverAddr,
		router:     router,
		Session:    "",
		connection: connection,
		status:     CLIENT_CREATED,
		timeout:    timeout,
	}

	err = ret.hello()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) hello() error {
	var err error
	if c.status != CLIENT_CREATED {
		return ErrClientIllegalOperation
	}

	start := time.Now().UnixMicro()

	var response *Response
	if response, err = c.Tell(Request{
		Identifier: "prt-hello",
	}); err != nil {
		return err
	}

	if response.Identifier != "prt-hello" {
		return ErrServerProcotol
	}

	c.RTT = time.Now().UnixMicro() - start
	c.Session = response.Session
	c.MaxLifeTime, err = strconv.ParseInt(response.Body, 10, 64)
	if err != nil {
		return ErrServerProcotol
	}

	c.status = CLIENT_ESTABLISHED
	c.Write(Request{
		Session:    c.Session,
		Identifier: "prt-established",
	})
	return nil
}

func (c *Client) Refresh() error

func (c *Client) AddRouter(identifier string, controller func(Request) *Response) bool {
	if strings.Index(identifier, "prt-") == 0 {
		return false
	}

	c.router[identifier] = controller
	return true
}

func (c *Client) DelSession()

// 向对方发送信息，并且期待 ACK
//
// 此函数会阻塞线程
func (c *Client) Tell(req Request) (*Response, error) {
	recvBuf := make([]byte, MAX_TRANSMIT_SIZE)
	var n int
	var err error

	reqBytes := req.ToBytes()
	if len(reqBytes) > MAX_TRANSMIT_SIZE {
		return nil, ErrContentOverflowed
	}

	c.connection.Write(req.ToBytes())

	ch := make(chan bool)
	go Timer(c.timeout, ch, false)

	go func(n *int, err *error, ch chan bool) {
		defer func() { recover() }()
		*n, *err = c.connection.Read(recvBuf)
		ch <- true
	}(&n, &err, ch)

	ok := <-ch
	close(ch)

	if !ok || n == 0 {
		return nil, ErrClientTellServerTimeout
	}

	return CastToResponse(recvBuf[:n])
}

// 向对方发送消息，但是不期待 ACK
func (c *Client) Write(req Request) {
	c.connection.Write(req.ToBytes())
}
