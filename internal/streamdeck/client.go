package streamdeck

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn       *websocket.Conn
	pluginUUID string
	writeMu    sync.Mutex
}

func Connect(port int, pluginUUID, registerEvent string) (*Client, error) {
	url := fmt.Sprintf("ws://127.0.0.1:%d", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, pluginUUID: pluginUUID}
	reg := map[string]string{"event": registerEvent, "uuid": pluginUUID}
	if err := c.writeJSON(reg); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) PluginUUID() string { return c.pluginUUID }

func (c *Client) ReadEvent() (Event, error) {
	var ev Event
	err := c.conn.ReadJSON(&ev)
	return ev, err
}

func (c *Client) writeJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}
