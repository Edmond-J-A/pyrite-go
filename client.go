package pyritego

import (
	"errors"
	"net"
	"strings"
	"time"
)

type Client struct {
	server     net.UDPAddr
	router     map[string]func(string) string
	connection *net.UDPConn

	session string
	timeout time.Duration

	sequence      int                      // 下一个 sequence
	promiseBuffer map[int]chan *PrtPackage // 暂存已发但未确认的包
}

var (
	ErrClientUDPBindingFailed = errors.New("udp binding failed")
	ErrServerProcotol         = errors.New("invalid server protocol")
)

func NewClient(serverAddr net.UDPAddr, timeout time.Duration) (*Client, error) {
	src := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	connection, err := net.DialUDP("udp", src, &serverAddr)
	if err != nil {
		return nil, ErrClientUDPBindingFailed
	}

	return &Client{
		server:     serverAddr,
		router:     make(map[string]func(string) string),
		connection: connection,
		timeout:    timeout,
		sequence:   0,
	}, nil
}

func (c *Client) getSequence() int {
	c.sequence += 1
	return c.sequence - 1
}

func (c *Client) Refresh() error

func (c *Client) AddRouter(identifier string, controller func(string) string) bool {
	if strings.Index(identifier, "prt-") == 0 {
		return false
	}

	c.router[identifier] = controller
	return true
}

// 向对方发送信息，并且期待 ACK
//
// 此函数会阻塞线程
func (c *Client) Promise(identifier, body string) (string, error) {
	req := PrtPackage{
		Session:    c.session,
		Identifier: identifier,
		sequence:   c.getSequence(),
		Body:       body,
	}

	reqBytes := req.ToBytes()
	if len(reqBytes) > MAX_TRANSMIT_SIZE {
		return "", ErrContentOverflowed
	}

	c.connection.Write(req.ToBytes())
	c.promiseBuffer[req.sequence] = make(chan *PrtPackage)

	ch := make(chan bool)
	go Timer(c.timeout, ch, false)

	var response *PrtPackage
	go func() {
		defer func() { recover() }()
		response = <-c.promiseBuffer[req.sequence]
		ch <- true
	}()

	ok := <-ch
	close(ch)
	if !ok {
		return "", ErrTimeout
	}

	return response.Body, nil
}

// 向对方发送消息，但是不期待 ACK
func (c *Client) Tell(identifier, body string) error {
	rBytes := PrtPackage{
		Session:    c.session,
		Identifier: identifier,
		sequence:   -1,
		Body:       body,
	}.ToBytes()

	if len(rBytes) > MAX_TRANSMIT_SIZE {
		return ErrContentOverflowed
	}

	c.connection.Write(rBytes)
	return nil
}

func (c *Client) processAck(response *PrtPackage) {
	ch, ok := c.promiseBuffer[response.sequence]
	if !ok {
		return
	}

	ch <- response
	close(ch)
	delete(c.promiseBuffer, response.sequence)
}

func (c *Client) process(recv []byte) {
	req, err := CastToPrtpackage(recv)
	if err != nil {
		return
	}

	if req.Identifier == "prt-ack" {
		c.processAck(req)
		return
	}

	f, ok := c.router[req.Identifier]
	if !ok {
		return
	}

	resp := f(req.Body)
	if resp == "" {
		return
	}

	c.connection.Write(PrtPackage{
		Session:    c.session,
		Identifier: "prt-ack",
		sequence:   req.sequence,
		Body:       resp,
	}.ToBytes())
}

func (c *Client) Start() {
	recvBuf := make([]byte, MAX_TRANSMIT_SIZE)
	var n int
	var err error
	for {
		n, err = c.connection.Read(recvBuf)
		if err != nil || n == 0 {
			panic("invalid msg recved")
		}

		slice := make([]byte, n)
		if copy(slice, recvBuf) != n {
			panic("invalid copy in client main loop")
		}

		go c.process(slice)
	}
}
