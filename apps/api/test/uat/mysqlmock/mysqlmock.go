// Package mysqlmock is a minimal MySQL wire-protocol server for UAT: it records what noXis sends and answers with a configured reply.
package mysqlmock

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// capability flags the mock advertises (protocol 4.1, secure connection, plugin auth, connect-with-db).
const capabilities uint32 = 0x0000_0001 | 0x0000_0200 | 0x0000_2000 | 0x0000_8000 | 0x0008_0000

// Server accepts MySQL connections and answers them from its configured reply.
type Server struct {
	ln net.Listener

	mu      sync.Mutex
	queries []string
	columns []string
	values  []string
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

// Close stops the listener.
func (s *Server) Close() error { return s.ln.Close() }

// Answer makes every query return one row built from columns and values.
func (s *Server) Answer(columns, values []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.columns, s.values, s.errMsg = columns, values, ""
}

// Fail makes every query answer with a MySQL error.
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

func (s *Server) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	if err := s.handshake(conn, r); err != nil {
		return
	}
	for {
		payload, seq, err := readPacket(r)
		if err != nil || len(payload) == 0 {
			return
		}
		switch payload[0] {
		case 0x01: // COM_QUIT
			return
		case 0x03: // COM_QUERY
			query := string(payload[1:])
			s.record(query)
			s.answer(conn, seq+1, query)
		case 0x0e: // COM_PING
			if err := writePacket(conn, seq+1, okPacket()); err != nil {
				return
			}
		default:
			if err := writePacket(conn, seq+1, errPacket(1047, "unsupported command")); err != nil {
				return
			}
		}
	}
}

// handshake sends HandshakeV10 and accepts any credentials, so the mock only models queries and replies.
func (s *Server) handshake(conn net.Conn, r *bufio.Reader) error {
	scramble := make([]byte, 20)
	payload := []byte{10}
	payload = append(payload, "8.0.0-mock"...)
	payload = append(payload, 0)               // server version terminator
	payload = append(payload, 1, 0, 0, 0)      // connection id
	payload = append(payload, scramble[:8]...) // auth plugin data, part 1
	payload = append(payload, 0)               // filler
	payload = append(payload, byte(capabilities&0xff), byte((capabilities>>8)&0xff))
	payload = append(payload, 33)         // charset utf8_general_ci
	payload = append(payload, 0x02, 0x00) // status: autocommit
	payload = append(payload, byte((capabilities>>16)&0xff), byte((capabilities>>24)&0xff))
	payload = append(payload, 21) // auth plugin data length
	payload = append(payload, make([]byte, 10)...)
	payload = append(payload, scramble[8:]...) // part 2 (12 bytes + terminator handled below)
	payload = append(payload, 0)
	payload = append(payload, "mysql_native_password"...)
	payload = append(payload, 0)
	if err := writePacket(conn, 0, payload); err != nil {
		return err
	}
	// the client's HandshakeResponse41 is read and accepted as-is
	if _, _, err := readPacket(r); err != nil {
		return err
	}
	return writePacket(conn, 2, okPacket())
}

// answer replies to one COM_QUERY with a one-column/one-row result set, an error, or an OK for an empty statement.
func (s *Server) answer(conn net.Conn, seq byte, query string) {
	s.mu.Lock()
	columns := append([]string(nil), s.columns...)
	values := append([]string(nil), s.values...)
	errMsg := s.errMsg
	s.mu.Unlock()

	if query == "" || query == ";" {
		_ = writePacket(conn, seq, okPacket())
		return
	}
	if errMsg != "" {
		_ = writePacket(conn, seq, errPacket(1146, errMsg))
		return
	}
	if len(columns) == 0 {
		columns = []string{"result"}
	}
	if len(values) == 0 {
		values = make([]string, len(columns))
	}
	if err := writePacket(conn, seq, []byte{byte(len(columns))}); err != nil {
		return
	}
	next := seq + 1
	for _, name := range columns {
		if err := writePacket(conn, next, columnDefinition(name)); err != nil {
			return
		}
		next++
	}
	if err := writePacket(conn, next, eofPacket()); err != nil {
		return
	}
	next++
	if err := writePacket(conn, next, rowPacket(values)); err != nil {
		return
	}
	next++
	_ = writePacket(conn, next, eofPacket())
}

func (s *Server) record(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, query)
}

func columnDefinition(name string) []byte {
	var p []byte
	p = append(p, lenEncString("def")...)
	p = append(p, lenEncString("")...)
	p = append(p, lenEncString("")...)
	p = append(p, lenEncString("")...)
	p = append(p, lenEncString(name)...)
	p = append(p, lenEncString("")...)
	p = append(p, 0x0c)                   // fixed-length fields
	p = append(p, 33, 0)                  // charset
	p = append(p, 0xff, 0xff, 0xff, 0xff) // column length
	p = append(p, 0xfd)                   // VAR_STRING
	p = append(p, 0x00, 0x00)             // flags
	p = append(p, 0x00)                   // decimals
	p = append(p, 0x00, 0x00)             // filler
	return p
}

func rowPacket(values []string) []byte {
	var p []byte
	for _, v := range values {
		p = append(p, lenEncString(v)...)
	}
	return p
}

func lenEncString(v string) []byte {
	if len(v) >= 251 {
		b := []byte{0xfc, byte(len(v)), byte(len(v) >> 8)}
		return append(b, v...)
	}
	return append([]byte{byte(len(v))}, v...)
}

func okPacket() []byte { return []byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00} }

func eofPacket() []byte { return []byte{0xfe, 0x00, 0x00, 0x02, 0x00} }

func errPacket(code uint16, message string) []byte {
	p := []byte{0xff}
	p = append(p, byte(code), byte(code>>8))
	p = append(p, '#')
	p = append(p, "42S02"...)
	return append(p, message...)
}

// readPacket reads one length-prefixed packet and its sequence number.
func readPacket(r *bufio.Reader) (payload []byte, seq byte, err error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, 0, err
	}
	length := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
	if length < 0 || length > 1<<24-1 {
		return nil, 0, errors.New("mysqlmock: bad packet length")
	}
	payload = make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, 0, err
	}
	return payload, header[3], nil
}

func writePacket(w io.Writer, seq byte, payload []byte) error {
	if len(payload) >= 1<<24 {
		return fmt.Errorf("mysqlmock: packet too large")
	}
	buf := make([]byte, 4+len(payload))
	buf[0], buf[1], buf[2], buf[3] = byte(len(payload)), byte(len(payload)>>8), byte(len(payload)>>16), seq
	copy(buf[4:], payload)
	_, err := w.Write(buf)
	return err
}
