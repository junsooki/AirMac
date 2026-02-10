package signaling

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Handler callbacks for incoming signaling messages.
type Handler struct {
	OnRegistered      func()
	OnOffer           func(from string, payload json.RawMessage)
	OnAnswer          func(from string, payload json.RawMessage)
	OnICECandidate    func(from string, payload json.RawMessage)
	OnHostsUpdated    func(hosts []HostInfo)
	OnHostDisconnected func(hostID string)
	OnError           func(msg string)
}

// Client is a WebSocket signaling client.
type Client struct {
	url        string
	clientID   string
	clientType string
	handler    Handler

	conn   *websocket.Conn
	mu     sync.Mutex
	done   chan struct{}
	closed bool
}

// NewClient creates a signaling client.
func NewClient(url, clientID, clientType string, handler Handler) *Client {
	return &Client{
		url:        url,
		clientID:   clientID,
		clientType: clientType,
		handler:    handler,
		done:       make(chan struct{}),
	}
}

// Connect dials the signaling server and starts reading messages.
func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("signaling dial: %w", err)
	}
	c.conn = conn

	// Register with the server.
	err = c.send(Message{
		Type:       TypeRegister,
		ID:         c.clientID,
		ClientType: c.clientType,
	})
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("signaling register: %w", err)
	}

	go c.readLoop()
	go c.pingLoop()
	return nil
}

// Close shuts down the connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

// SendOffer sends an SDP offer to target.
func (c *Client) SendOffer(target string, payload json.RawMessage) error {
	return c.send(Message{Type: TypeOffer, Target: target, Payload: payload})
}

// SendAnswer sends an SDP answer to target.
func (c *Client) SendAnswer(target string, payload json.RawMessage) error {
	return c.send(Message{Type: TypeAnswer, Target: target, Payload: payload})
}

// SendICECandidate sends an ICE candidate to target.
func (c *Client) SendICECandidate(target string, payload json.RawMessage) error {
	return c.send(Message{Type: TypeICECandidate, Target: target, Payload: payload})
}

// RequestHostList asks the server for available hosts.
func (c *Client) RequestHostList() error {
	return c.send(Message{Type: TypeListHosts})
}

func (c *Client) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.WriteJSON(msg)
}

func (c *Client) readLoop() {
	defer c.Close()
	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				log.Printf("signaling read error: %v", err)
				return
			}
		}
		c.dispatch(msg)
	}
}

func (c *Client) dispatch(msg Message) {
	switch msg.Type {
	case TypeRegistered:
		if c.handler.OnRegistered != nil {
			c.handler.OnRegistered()
		}
	case TypeOffer:
		if c.handler.OnOffer != nil {
			c.handler.OnOffer(msg.From, msg.Payload)
		}
	case TypeAnswer:
		if c.handler.OnAnswer != nil {
			c.handler.OnAnswer(msg.From, msg.Payload)
		}
	case TypeICECandidate:
		if c.handler.OnICECandidate != nil {
			c.handler.OnICECandidate(msg.From, msg.Payload)
		}
	case TypeHosts, TypeHostsUpdated:
		if c.handler.OnHostsUpdated != nil {
			c.handler.OnHostsUpdated(msg.List)
		}
	case TypeHostDisconnected:
		if c.handler.OnHostDisconnected != nil {
			c.handler.OnHostDisconnected(msg.HostID)
		}
	case TypeError:
		if c.handler.OnError != nil {
			c.handler.OnError(msg.Msg)
		}
	case TypePong:
		// heartbeat response, nothing to do
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			_ = c.send(Message{Type: TypePing})
		}
	}
}
