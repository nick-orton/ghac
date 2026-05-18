package snapcast

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Client is a SnapCast JSON-RPC 2.0 client over TCP. Commands and
// server-initiated notifications are multiplexed on a single connection.
// A reader goroutine dispatches responses to pending callers and forwards
// notifications to an internal channel.
type Client struct {
	conn    net.Conn
	writeMu sync.Mutex   // guards enc
	enc     *json.Encoder

	mu      sync.Mutex
	pending map[int64]chan json.RawMessage

	notify chan json.RawMessage
	nextID atomic.Int64
}

// rpcRequest is a JSON-RPC 2.0 request message sent to SnapCast.
type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

// rpcMessage is a parsed JSON-RPC 2.0 message received from SnapCast.
// It may be a response (ID non-nil) or a notification (ID nil, Method set).
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Method  string          `json:"method"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Params json.RawMessage `json:"params"`
}

// Connect dials SnapCast at addr (e.g. "192.168.1.10:1705"), starts the
// reader goroutine, and returns a ready-to-use Client. The caller must call
// Close() when done.
func Connect(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("snapcast connect: %w", err)
	}
	c := &Client{
		conn:    conn,
		enc:     json.NewEncoder(conn),
		pending: make(map[int64]chan json.RawMessage),
		notify:  make(chan json.RawMessage, 16),
	}
	go c.readLoop()
	return c, nil
}

// Close closes the TCP connection, which causes readLoop to exit and the
// notify channel to be closed.
func (c *Client) Close() {
	_ = c.conn.Close()
}

// readLoop reads JSON-RPC messages from the connection and dispatches them:
// responses go to the waiting caller's channel; notifications go to c.notify.
// Runs until the connection is closed; closes c.notify on exit.
func (c *Client) readLoop() {
	defer close(c.notify)
	dec := json.NewDecoder(c.conn)
	for {
		var msg rpcMessage
		if err := dec.Decode(&msg); err != nil {
			// Connection closed or read error — unblock all pending callers.
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}

		if msg.ID != nil {
			// Response to a pending request.
			c.mu.Lock()
			ch, ok := c.pending[*msg.ID]
			if ok {
				delete(c.pending, *msg.ID)
			}
			c.mu.Unlock()
			if ok {
				select {
				case ch <- msg.Result:
				default:
				}
			}
		} else if msg.Method != "" {
			// Server-initiated notification.
			select {
			case c.notify <- msg.Params:
			default:
				// Buffer full — drop. The next notification will still
				// trigger a full GetServerStatus re-fetch.
			}
		}
	}
}

// call sends a JSON-RPC 2.0 request and blocks until the response arrives,
// up to 5 seconds.
func (c *Client) call(method string, params interface{}) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	c.writeMu.Lock()
	err := c.enc.Encode(req)
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("snapcast send %s: %w", method, err)
	}

	select {
	case result, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("snapcast connection closed waiting for %s", method)
		}
		return result, nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("snapcast %s: request timed out", method)
	}
}

// serverStatusResult mirrors the relevant parts of the Server.GetStatus JSON
// response from SnapCast.
type serverStatusResult struct {
	Server struct {
		Groups []struct {
			Clients []struct {
				Config struct {
					Name   string `json:"name"`
					Volume struct {
						Muted   bool `json:"muted"`
						Percent int  `json:"percent"`
					} `json:"volume"`
				} `json:"config"`
				Host struct {
					Name string `json:"name"`
				} `json:"host"`
				ID        string `json:"id"`
				Connected bool   `json:"connected"`
			} `json:"clients"`
		} `json:"groups"`
	} `json:"server"`
}

// GetServerStatus fetches the full server status and returns all known
// SnapCast clients across all groups.
func (c *Client) GetServerStatus() ([]SnapClient, error) {
	raw, err := c.call("Server.GetStatus", nil)
	if err != nil {
		return nil, err
	}

	var result serverStatusResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("snapcast GetServerStatus parse: %w", err)
	}

	var clients []SnapClient
	for _, group := range result.Server.Groups {
		for _, rc := range group.Clients {
			name := rc.Config.Name
			if name == "" {
				name = rc.Host.Name
			}
			clients = append(clients, SnapClient{
				ID:     rc.ID,
				Name:   name,
				Volume: rc.Config.Volume.Percent,
				Muted:  rc.Config.Volume.Muted,
			})
		}
	}
	return clients, nil
}

// setVolumeParams is the JSON parameter structure for Client.SetVolume.
type setVolumeParams struct {
	ID     string `json:"id"`
	Volume struct {
		Muted   bool `json:"muted"`
		Percent int  `json:"percent"`
	} `json:"volume"`
}

// SetVolume sets both the volume (0–100) and muted state for a client in a
// single RPC call. The SnapCast protocol encodes them together, so both values
// must always be supplied.
func (c *Client) SetVolume(clientID string, vol int, muted bool) error {
	var p setVolumeParams
	p.ID = clientID
	p.Volume.Percent = vol
	p.Volume.Muted = muted
	_, err := c.call("Client.SetVolume", p)
	return err
}

// SetMute toggles only the muted flag for a client. currentVol must be the
// client's present volume so the SnapCast protocol receives a complete update.
func (c *Client) SetMute(clientID string, muted bool, currentVol int) error {
	return c.SetVolume(clientID, currentVol, muted)
}

// setNameParams is the JSON parameter structure for Client.SetName.
type setNameParams struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SetName sets the display name for a SnapCast client.
func (c *Client) SetName(clientID, name string) error {
	p := setNameParams{ID: clientID, Name: name}
	_, err := c.call("Client.SetName", p)
	return err
}

// ListenNotifications returns a tea.Cmd that blocks until the next SnapCast
// server notification arrives, fetches the updated server status, and returns
// a MsgClientsUpdated (or MsgError on failure). Call the returned Cmd again
// from Update to keep listening — the same re-subscribe pattern used by
// mpd.Client.ListenIdle.
func (c *Client) ListenNotifications() tea.Cmd {
	return func() tea.Msg {
		_, ok := <-c.notify
		if !ok {
			return MsgError{Err: fmt.Errorf("snapcast connection closed")}
		}
		clients, err := c.GetServerStatus()
		if err != nil {
			return MsgError{Err: fmt.Errorf("snapcast status after notification: %w", err)}
		}
		return MsgClientsUpdated{Clients: clients}
	}
}
