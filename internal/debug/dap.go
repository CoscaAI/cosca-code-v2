// Package debug é o COSCA DEBUG ENGINE (spec seções 20-22): um host DAP
// universal. Fala o Debug Adapter Protocol sobre TCP com qualquer debug
// adapter (dlv para Go, debugpy para Python, etc.). Nunca um debugger
// artificial — a mecânica real de debug fica com o adapter nativo; aqui é o
// contrato (padrão broker/adapter da pesquisa).
package debug

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// request / response / event são as mensagens JSON-RPC do DAP.
type request struct {
	Seq     int             `json:"seq"`
	Type    string          `json:"type"` // "request"
	Command string          `json:"command"`
	Args    json.RawMessage `json:"arguments,omitempty"`
}

type response struct {
	Seq     int             `json:"seq"`
	Type    string          `json:"type"` // "response"
	ReqSeq  int             `json:"request_seq"`
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`
}

type eventMsg struct {
	Seq   int             `json:"seq"`
	Type  string          `json:"type"` // "event"
	Event string          `json:"event"`
	Body  json.RawMessage `json:"body,omitempty"`
}

// Event é um evento DAP (stopped, terminated, output...) entregue ao cliente.
type Event struct {
	Name string
	Body json.RawMessage
}

// Client é um cliente DAP sobre TCP.
type Client struct {
	conn    net.Conn
	reader  *bufio.Reader
	mu      sync.Mutex
	nextID  int
	pending map[int]chan *response
	events  chan Event
}

// Dial conecta a um debug adapter DAP em addr (host:port).
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		pending: map[int]chan *response{},
		events:  make(chan Event, 64),
	}
	go c.readLoop()
	return c, nil
}

// Close fecha a conexão.
func (c *Client) Close() { _ = c.conn.Close() }

// Events devolve o canal de eventos (stopped/terminated/output).
func (c *Client) Events() <-chan Event { return c.events }

// request envia um request DAP e espera a resposta.
func (c *Client) request(command string, args any) (json.RawMessage, error) {
	raw, _ := json.Marshal(args)
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan *response, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	msg, _ := json.Marshal(request{Seq: id, Type: "request", Command: command, Args: raw})
	if err := c.write(msg); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("sem resposta para %s", command)
		}
		if !resp.Success {
			return nil, fmt.Errorf("%s falhou: %s", command, resp.Message)
		}
		return resp.Body, nil
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("timeout em %s", command)
	}
}

// write envia uma mensagem com framing Content-Length.
func (c *Client) write(msg []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(msg))
	if _, err := c.conn.Write([]byte(header)); err != nil {
		return err
	}
	_, err := c.conn.Write(msg)
	return err
}

// readLoop lê mensagens do adapter e faz dispatch.
func (c *Client) readLoop() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			close(c.events)
			return
		}
		var resp response
		if json.Unmarshal(msg, &resp) == nil && resp.Type == "response" {
			c.mu.Lock()
			ch, ok := c.pending[resp.ReqSeq]
			if ok {
				ch <- &resp
				delete(c.pending, resp.ReqSeq)
			}
			c.mu.Unlock()
			continue
		}
		var ev eventMsg
		if json.Unmarshal(msg, &ev) == nil && ev.Type == "event" {
			c.events <- Event{Name: ev.Event, Body: ev.Body}
		}
	}
}

// readMessage lê uma mensagem com framing Content-Length.
func (c *Client) readMessage() ([]byte, error) {
	var contentLength int
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(v)
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("Content-Length inválido")
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
