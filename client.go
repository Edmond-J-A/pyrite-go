package pyritego

import "net"

type Client struct {
	server  net.UDPAddr
	router  map[string]func(Request) Response
	Session string
}

func NewClient(serverAddr net.UDPAddr) (Client, error)
func SayHello() error
func (c *Client) AddRouter(identifier string, controller func(Request) Response)
func (c *Client) DelSession()
func (c *Client) Tell(identifier, body string) (Response, error)
