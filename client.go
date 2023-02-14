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

	sequence     int                    // 下一个 sequence
	sequenceBuff map[int]chan *Response // 暂存已发但未确认的包
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
		sequence:   0,
	}

	err = ret.hello()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) getSequence() int {
	c.sequence += 1
	return c.sequence - 1
}

func (c *Client) hello() error {
	var err error
	if c.status != CLIENT_CREATED {
		return ErrClientIllegalOperation
	}

	start := time.Now().UnixMicro()

	var response *Response
	if response, err = c.Tell("prt-hello", ""); err != nil {
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
	c.Write("prt-established", "")
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
func (c *Client) Tell(identifier, body string) (*Response, error) {
	var response *Response
	var err error
	req := Request{
		Session:    c.Session,
		Identifier: identifier,
		sequence:   c.getSequence(),
		Body:       body,
	}

	reqBytes := req.ToBytes()
	if len(reqBytes) > MAX_TRANSMIT_SIZE {
		return nil, ErrContentOverflowed
	}

	c.connection.Write(req.ToBytes())
	c.sequenceBuff[req.sequence] = make(chan *Response)

	ch := make(chan bool)
	go Timer(c.timeout, ch, false)

	go func(err *error, ch chan bool) {
		defer func() { recover() }()
		response = <-c.sequenceBuff[req.sequence]
		ch <- true
	}(&err, ch)

	ok := <-ch
	close(ch)
	if !ok {
		return nil, ErrClientTellServerTimeout
	}

	return response, nil
}

// 向对方发送消息，但是不期待 ACK
func (c *Client) Write(identifier, body string) {
	c.connection.Write(Request{
		Session:    c.Session,
		Identifier: identifier,
		sequence:   -1,
		Body:       body,
	}.ToBytes())
}

func (c *Client) processAck(response *Response) {
	ch, ok := c.sequenceBuff[response.sequence]
	if !ok {
		return
	}

	ch <- response
	close(ch)
	delete(c.sequenceBuff, response.sequence)
}

func (c *Client) process(recv []byte) {
	response, err := CastToResponse(recv)
	if err != nil {
		return
	}

	if response.Identifier == "prt-ack" {
		c.processAck(response)
		return
	}

	request, err := CastToRequest(recv)
	if err != nil {
		return
	}

	f, ok := c.router[request.Identifier]
	if !ok {
		return
	}

	resp := f(*request)
	if resp == nil {
		return
	}

	resp.Identifier = "prt-ack"
	c.connection.Write(resp.ToBytes())
}

func (c *Client) Start() {
	if c.status != CLIENT_ESTABLISHED {
		panic("invalid client status")
	}

	recvBuf := make([]byte, MAX_TRANSMIT_SIZE)
	var n int
	var err error
	for {
		n, err = c.connection.Read(recvBuf)
		if err != nil || n == 0 {
			panic("invalid msg recved")
		}

		go c.process(recvBuf[:n])
	}
}
