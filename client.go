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
}

var (
	ErrClientIllegalOperation = errors.New("illegal operation")
	ErrClientUDPBindingFailed = errors.New("udp binding failed")
	ErrContentOverflowed      = errors.New("content overflowed")
	ErrServerProcotol         = errors.New("invalid server protocol")
)

func NewClient(serverAddr net.UDPAddr) (*Client, error) {
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
	}

	err = ret.hello()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) hello() error {
	if c.status != CLIENT_CREATED {
		return ErrClientIllegalOperation
	}

	msg := Request{
		Identifier: "prt-hello",
	}.ToBytes()

	if len(msg) > MAX_TRANSMIT_SIZE {
		return ErrContentOverflowed
	}

	start := time.Now().UnixMicro()

	c.connection.Write(msg)
	recvBuf := make([]byte, MAX_TRANSMIT_SIZE)
	n, err := c.connection.Read(recvBuf)
	if err != nil {
		return err
	}

	c.RTT = time.Now().UnixMicro() - start

	response, err := CastToResponse(recvBuf[:n])
	if err != nil {
		return err
	}

	if response.Identifier != "prt-hello" {
		return ErrServerProcotol
	}

	c.Session = response.Session
	c.MaxLifeTime, err = strconv.ParseInt(response.Body, 10, 64)
	if err != nil {
		return ErrServerProcotol
	}

	msg = Request{
		Session:    c.Session,
		Identifier: "prt-established",
	}.ToBytes()

	c.connection.Write(msg)
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
func (c *Client) Tell(identifier, body string) (Response, error)
