// Package pgmock is a minimal PostgreSQL wire-protocol server for UAT: it records what noXis sends and answers with a configured reply.
package pgmock

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
)

// Server accepts PostgreSQL connections and answers them from its configured reply.
type Server struct {
	ln net.Listener

	mu      sync.Mutex
	queries []string
	columns []string
	rows    [][]string
	errMsg  string
}

// Start listens on every interface so a container can reach it through the host gateway.
func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln}
	go s.accept()
	return s, nil
}

// Port is the listener's TCP port.
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// Close stops the listener; open connections end with their client.
func (s *Server) Close() error { return s.ln.Close() }

// Answer makes every query return one row built from columns and values.
func (s *Server) Answer(columns []string, row []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.columns, s.rows, s.errMsg = columns, [][]string{row}, ""
}

// Fail makes every query answer with a PostgreSQL error.
func (s *Server) Fail(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errMsg = message
}

// Queries is the ledger of query strings received so far.
func (s *Server) Queries() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.queries...)
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

// serve speaks enough of the simple-query protocol for pgx to connect, ping and read one result.
func (s *Server) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	if err := s.handshake(conn, r); err != nil {
		return
	}
	for {
		kind, body, err := readMessage(r)
		if err != nil {
			return
		}
		switch kind {
		case 'X': // terminate
			return
		case 'Q':
			s.record(trimNUL(body))
			s.answer(conn, trimNUL(body))
		default:
			// the simple protocol only carries Q; ignore anything else rather than guess
		}
	}
}

// handshake answers the startup (and the SSL probe that pgx sends first with sslmode=prefer).
func (s *Server) handshake(conn net.Conn, r *bufio.Reader) error {
	for {
		var length int32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			return err
		}
		if length < 8 {
			return errors.New("pgmock: short startup message")
		}
		body := make([]byte, length-4)
		if _, err := io.ReadFull(r, body); err != nil {
			return err
		}
		code := binary.BigEndian.Uint32(body)
		if code == 80877103 { // SSLRequest: refuse and let the client continue in plaintext
			if _, err := conn.Write([]byte{'N'}); err != nil {
				return err
			}
			continue
		}
		break
	}
	// AuthenticationOk, then the parameter/backend messages pgx expects before ReadyForQuery.
	if err := writeMessage(conn, 'R', i32(0)); err != nil {
		return err
	}
	for _, kv := range [][2]string{
		{"server_version", "16.0"},
		{"client_encoding", "UTF8"},
		{"DateStyle", "ISO, MDY"},
		{"integer_datetimes", "on"},
		{"standard_conforming_strings", "on"},
	} {
		payload := append([]byte(kv[0]), 0)
		payload = append(payload, append([]byte(kv[1]), 0)...)
		if err := writeMessage(conn, 'S', payload); err != nil {
			return err
		}
	}
	if err := writeMessage(conn, 'K', append(i32(1234), i32(5678)...)); err != nil {
		return err
	}
	return writeMessage(conn, 'Z', []byte{'I'})
}

// answer replies to one query: an empty query, the configured error, or the configured row.
func (s *Server) answer(conn net.Conn, query string) {
	s.mu.Lock()
	columns := append([]string(nil), s.columns...)
	rows := append([][]string(nil), s.rows...)
	errMsg := s.errMsg
	s.mu.Unlock()

	if strings.Trim(strings.TrimSpace(query), "; ") == "" {
		if err := writeMessage(conn, 'I', nil); err != nil {
			return
		}
		_ = writeMessage(conn, 'Z', []byte{'I'})
		return
	}
	if errMsg != "" {
		payload := []byte{'S'}
		payload = appendCString(payload, "ERROR")
		payload = append(payload, 'M')
		payload = appendCString(payload, errMsg)
		payload = append(payload, 0)
		if err := writeMessage(conn, 'E', payload); err != nil {
			return
		}
		_ = writeMessage(conn, 'Z', []byte{'I'})
		return
	}
	if err := writeMessage(conn, 'T', rowDescription(columns)); err != nil {
		return
	}
	for _, row := range rows {
		if err := writeMessage(conn, 'D', dataRow(row)); err != nil {
			return
		}
	}
	if err := writeMessage(conn, 'C', appendCString(nil, "SELECT "+itoa(len(rows)))); err != nil {
		return
	}
	_ = writeMessage(conn, 'Z', []byte{'I'})
}

func (s *Server) record(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, query)
}

// rowDescription describes every column as text with unknown length.
func rowDescription(columns []string) []byte {
	payload := i16(int16(len(columns)))
	for _, name := range columns {
		payload = appendCString(payload, name)
		payload = append(payload, i32(0)...)  // table oid
		payload = append(payload, i16(0)...)  // column number
		payload = append(payload, i32(25)...) // text
		payload = append(payload, i16(-1)...) // type length
		payload = append(payload, i32(-1)...) // type modifier
		payload = append(payload, i16(0)...)  // text format
	}
	return payload
}

func dataRow(row []string) []byte {
	payload := i16(int16(len(row)))
	for _, value := range row {
		payload = append(payload, i32(int32(len(value)))...)
		payload = append(payload, value...)
	}
	return payload
}

func readMessage(r *bufio.Reader) (kind byte, body []byte, err error) {
	kind, err = r.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return 0, nil, err
	}
	if length < 4 {
		return 0, nil, errors.New("pgmock: short message")
	}
	body = make([]byte, length-4)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, err
	}
	return kind, body, nil
}

func writeMessage(w io.Writer, kind byte, payload []byte) error {
	buf := make([]byte, 5+len(payload))
	buf[0] = kind
	binary.BigEndian.PutUint32(buf[1:], uint32(4+len(payload)))
	copy(buf[5:], payload)
	_, err := w.Write(buf)
	return err
}

func appendCString(dst []byte, s string) []byte {
	dst = append(dst, s...)
	return append(dst, 0)
}

func trimNUL(body []byte) string { return strings.TrimRight(string(body), "\x00") }

func i16(v int16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func i32(v int32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
