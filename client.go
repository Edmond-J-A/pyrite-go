package pyritego

import (
	"net"
)

type Client struct {
	server  net.UDPAddr
	router  map[string]func(Request) Response
	Session string

	connection *net.UDPConn
}

func NewClient(serverAddr net.UDPAddr) (*Client, error) {
	src := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	connection, err := net.DialUDP("udp", src, &serverAddr)
	if err != nil {
		return nil, ErrUDPClientBindingFailed
	}

	router := make(map[string]func(Request) Response)
	ret := &Client{
		server:     serverAddr,
		router:     router,
		Session:    "",
		connection: connection,
	}

	if err = ret.hello(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (c *Client) hello() error
func (c *Client) Refresh() error
func (c *Client) AddRouter(identifier string, controller func(Request) Response)
func (c *Client) DelSession()
func (c *Client) Tell(identifier, body string) (Response, error)
