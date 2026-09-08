package realtime

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
	pingEvery = 25 * time.Second
)

type Client struct {
	gameID    string
	playerID  string
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newClient(gameID, playerID string, conn *websocket.Conn) *Client {
	return &Client{
		gameID: gameID, playerID: playerID, conn: conn,
		send: make(chan []byte, 32), done: make(chan struct{}),
	}
}

func (c *Client) run(hub *Hub) {
	if err := hub.Register(c); err != nil {
		c.close()
		return
	}
	defer hub.Unregister(c)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		c.writePump()
	}()
	c.readPump()
	c.close()
	<-writerDone
}

func (c *Client) enqueue(payload []byte) {
	select {
	case <-c.done:
		return
	case c.send <- payload:
		return
	default:
		c.close()
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "connection closed"),
				time.Now().Add(writeWait),
			)
			_ = c.conn.Close()
		}
	})
}

func (c *Client) readPump() {
	if c.conn == nil {
		return
	}
	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	if c.conn == nil {
		return
	}
	ticker := time.NewTicker(pingEvery)
	defer ticker.Stop()
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			deadline := time.Now().Add(writeWait)
			if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return
			}
		}
	}
}
