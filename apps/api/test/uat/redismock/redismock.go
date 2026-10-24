// Package redismock is a minimal RESP server for UAT: it records the commands noXis sends and answers browse/read commands from a configured reply.
package redismock

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Server accepts Redis-protocol connections and answers the commands noXis uses.
type Server struct {
	ln net.Listener

	mu       sync.Mutex
	keys     []string
	value    string
	ttlMs    int64
	errMsg   string
	commands []string
}

// Start listens on every interface so a container can reach it through the host gateway.
func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, ttlMs: 600_000}
	go s.accept()
	return s, nil
}

// Port is the listener's TCP port.
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// Close stops the listener.
func (s *Server) Close() error { return s.ln.Close() }

// Answer sets the keys a SCAN returns and the value a GET returns.
func (s *Server) Answer(keys []string, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys, s.value, s.errMsg = keys, value, ""
}

// Fail makes every data command answer with an error.
func (s *Server) Fail(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errMsg = message
}

// Commands is the ledger of commands received so far, verb and arguments.
func (s *Server) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

func (s *Server) accept() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.serve(conn)
	}
}

func (s *Server) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}
		s.record(args)
		if err := s.reply(w, args); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
	}
}

func (s *Server) reply(w *bufio.Writer, args []string) error {
	verb := strings.ToUpper(args[0])
	s.mu.Lock()
	errMsg := s.errMsg
	keys := append([]string(nil), s.keys...)
	value := s.value
	ttl := s.ttlMs
	s.mu.Unlock()

	switch verb {
	case "PING":
		_, err := w.WriteString("+PONG\r\n")
		return err
	case "HELLO":
		// refusing HELLO makes the client fall back to RESP2, which is all this mock speaks
		_, err := w.WriteString("-ERR unknown command 'HELLO'\r\n")
		return err
	case "AUTH", "SELECT", "CLIENT", "SETINFO", "FLUSHDB", "FLUSHALL":
		_, err := w.WriteString("+OK\r\n")
		return err
	case "COMMAND":
		_, err := w.WriteString("*0\r\n")
		return err
	}
	if errMsg != "" {
		_, err := w.WriteString("-ERR " + errMsg + "\r\n")
		return err
	}
	switch verb {
	case "SCAN":
		return writeArray(w, []string{"0", ""}, map[string][]string{"": keys})
	case "KEYS":
		return writeBulkArray(w, keys)
	case "TYPE":
		_, err := w.WriteString("+string\r\n")
		return err
	case "GET":
		return writeBulk(w, value)
	case "STRLEN":
		_, err := w.WriteString(":" + strconv.Itoa(len(value)) + "\r\n")
		return err
	case "PTTL", "TTL":
		_, err := w.WriteString(":" + strconv.FormatInt(ttl, 10) + "\r\n")
		return err
	case "LRANGE":
		return writeBulkArray(w, []string{value})
	case "HGETALL":
		return writeBulkArray(w, []string{"field", value})
	case "DEL", "UNLINK", "EXISTS":
		_, err := w.WriteString(":" + strconv.Itoa(len(args)-1) + "\r\n")
		return err
	}
	_, err := w.WriteString("+OK\r\n")
	return err
}

func (s *Server) record(args []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, strings.Join(args, " "))
}

// readCommand reads one RESP array or inline command.
func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return nil, nil
	}
	if line[0] != '*' {
		return strings.Fields(line), nil
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		header = strings.TrimRight(header, "\r\n")
		if len(header) == 0 || header[0] != '$' {
			return nil, errors.New("redismock: expected a bulk string")
		}
		size, err := strconv.Atoi(header[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

// writeArray writes a two-element array whose second element is a key list, matching SCAN's shape.
func writeArray(w *bufio.Writer, head []string, lists map[string][]string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(head)); err != nil {
		return err
	}
	if err := writeBulk(w, head[0]); err != nil {
		return err
	}
	return writeBulkArray(w, lists[""])
}

func writeBulkArray(w *bufio.Writer, values []string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(values)); err != nil {
		return err
	}
	for _, v := range values {
		if err := writeBulk(w, v); err != nil {
			return err
		}
	}
	return nil
}

func writeBulk(w *bufio.Writer, value string) error {
	if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(value), value); err != nil {
		return err
	}
	return nil
}
