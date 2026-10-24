// Package mongomock is a minimal MongoDB wire-protocol server for UAT: it records the commands noXis sends and answers find/list commands from a configured reply.
package mongomock

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/pbkdf2"
)

// opMsg and opQuery are the MongoDB opcodes this mock understands.
const (
	opMsg   int32 = 2013
	opQuery int32 = 2004
)

// Server accepts MongoDB connections and answers handshake, find and list commands.
type Server struct {
	ln net.Listener

	mu          sync.Mutex
	rows        bson.A
	errMsg      string
	commands    []string
	filters     []bson.M
	password    string
	serverSalt  []byte
	authMessage string
}

// Start listens on every interface so a container can reach it through the host gateway.
func Start() (*Server, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, password: "mock", serverSalt: []byte("noxis-uat-salt!!")}
	go s.accept()
	return s, nil
}

// Port is the listener's TCP port.
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// Close stops the listener.
func (s *Server) Close() error { return s.ln.Close() }

// Answer makes a find return the given documents.
func (s *Server) Answer(rows ...bson.M) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = make(bson.A, len(rows))
	for i := range rows {
		s.rows[i] = rows[i]
	}
	s.errMsg = ""
}

// Fail makes a find answer with a MongoDB error.
func (s *Server) Fail(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errMsg = message
}

// Commands is the ledger of command names received so far.
func (s *Server) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

// Filters is the ledger of find filters received so far.
func (s *Server) Filters() []bson.M {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]bson.M(nil), s.filters...)
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
	for {
		requestID, opCode, body, err := readMessage(conn)
		if err != nil {
			return
		}
		if opCode == opQuery {
			// legacy handshake: flags(4) + cstring + cstring + numberToSkip(4) + numberToReturn(4) + document
			if doc, err := bson.Marshal(handshake()); err == nil {
				if err := writeQueryReply(conn, requestID, doc); err != nil {
					return
				}
			}
			continue
		}
		if opCode != opMsg || len(body) < 5 {
			return
		}
		// flags (4) + section kind 0 (1) + a BSON document
		raw := bson.Raw(body[5:])
		name := commandName(raw)
		s.record(name, raw)
		reply := s.reply(name, raw)
		out, err := bson.Marshal(reply)
		if err != nil {
			return
		}
		if err := writeMessage(conn, requestID, out); err != nil {
			return
		}
	}
}

// commandName is the first key of the command document.
func commandName(raw bson.Raw) string {
	elems, err := raw.Elements()
	if err != nil || len(elems) == 0 {
		return ""
	}
	return elems[0].Key()
}

func (s *Server) record(name string, raw bson.Raw) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, name)
	if name == "find" {
		var filter bson.M
		if v, err := raw.LookupErr("filter"); err == nil {
			_ = v.Unmarshal(&filter)
		}
		s.filters = append(s.filters, filter)
	}
}

func (s *Server) reply(name string, raw bson.Raw) bson.M {
	s.mu.Lock()
	rows := append(bson.A(nil), s.rows...)
	errMsg := s.errMsg
	s.mu.Unlock()

	switch name {
	case "hello", "ismaster", "isMaster":
		return handshake()
	case "ping", "endSessions", "killCursors", "getParameter", "buildInfo":
		return bson.M{"ok": 1}
	case "saslStart":
		return s.saslStart(raw)
	case "saslContinue":
		return s.saslContinue(raw)
	case "listDatabases":
		return bson.M{"ok": 1, "databases": bson.A{
			bson.M{"name": "admin", "sizeOnDisk": int64(1), "empty": false},
			bson.M{"name": "demo", "sizeOnDisk": int64(1), "empty": false},
		}}
	case "listCollections":
		ns := databaseOf(raw) + ".$cmd.listCollections"
		return cursorReply(ns, bson.A{bson.M{"name": "items", "type": "collection"}})
	case "find":
		if errMsg != "" {
			return bson.M{"ok": 0, "errmsg": errMsg, "code": 26}
		}
		collection, _ := raw.Lookup("find").StringValueOK()
		return cursorReply(databaseOf(raw)+"."+collection, rows)
	}
	return bson.M{"ok": 1}
}

// saslStart answers SCRAM-SHA-1 with a server-first message; the proof is not verified, only the signature is completed.
func (s *Server) saslStart(raw bson.Raw) bson.M {
	_, payload := raw.Lookup("payload").Binary()
	clientFirst := string(payload)
	if i := strings.Index(clientFirst, "n,,"); i >= 0 {
		clientFirst = clientFirst[i+3:]
	}
	nonce := ""
	if i := strings.Index(clientFirst, "r="); i >= 0 {
		nonce = strings.SplitN(clientFirst[i+2:], ",", 2)[0]
	}
	serverFirst := "r=" + nonce + "mocknonce,s=" + base64.StdEncoding.EncodeToString(s.serverSalt) + ",i=4096"
	s.mu.Lock()
	s.authMessage = clientFirst + "," + serverFirst
	s.mu.Unlock()
	return bson.M{"ok": 1, "conversationId": 1, "done": false, "payload": bson.Binary{Data: []byte(serverFirst)}}
}

// saslContinue completes the conversation with the server signature the driver verifies.
func (s *Server) saslContinue(raw bson.Raw) bson.M {
	_, payload := raw.Lookup("payload").Binary()
	clientFinal := string(payload)
	withoutProof := clientFinal
	if i := strings.Index(clientFinal, ",p="); i >= 0 {
		withoutProof = clientFinal[:i]
	}
	s.mu.Lock()
	password := s.password
	salt := s.serverSalt
	authMessage := s.authMessage
	s.mu.Unlock()

	// MongoDB's SCRAM-SHA-1 feeds PBKDF2 the legacy MONGODB-CR digest, not the raw password
	digest := md5.Sum([]byte("mock:mongo:" + password))
	salted := pbkdf2.Key([]byte(hex.EncodeToString(digest[:])), salt, 4096, sha1.Size, sha1.New)
	serverKey := hmacSHA1(salted, []byte("Server Key"))
	signature := hmacSHA1(serverKey, []byte(authMessage+","+withoutProof))
	serverFinal := "v=" + base64.StdEncoding.EncodeToString(signature)
	return bson.M{"ok": 1, "conversationId": 1, "done": true, "payload": bson.Binary{Data: []byte(serverFinal)}}
}

func hmacSHA1(key, message []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// handshake is the OP_MSG hello reply shared by both handshake paths.
func handshake() bson.M {
	return bson.M{
		"ok": 1, "isWritablePrimary": true, "ismaster": true, "helloOk": true,
		"maxWireVersion": 21, "minWireVersion": 0,
		"maxBsonObjectSize": 16777216, "maxMessageSizeBytes": 48000000, "maxWriteBatchSize": 100000,
		"logicalSessionTimeoutMinutes": 30, "connectionId": 1,
	}
}

func databaseOf(raw bson.Raw) string {
	db, _ := raw.Lookup("$db").StringValueOK()
	if db == "" {
		return "demo"
	}
	return db
}

func cursorReply(ns string, batch bson.A) bson.M {
	if batch == nil {
		batch = bson.A{}
	}
	return bson.M{"ok": 1, "cursor": bson.M{"id": int64(0), "ns": ns, "firstBatch": batch}}
}

// readMessage reads one wire message and returns its request id, opcode and body.
func readMessage(r io.Reader) (requestID int32, opCode int32, body []byte, err error) {
	var header [16]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, 0, nil, err
	}
	length := int32(binary.LittleEndian.Uint32(header[0:4]))
	if length < 16 || length > 48_000_000 {
		return 0, 0, nil, errors.New("mongomock: bad message length")
	}
	requestID = int32(binary.LittleEndian.Uint32(header[4:8]))
	opCode = int32(binary.LittleEndian.Uint32(header[12:16]))
	body = make([]byte, length-16)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, 0, nil, err
	}
	return requestID, opCode, body, nil
}

// writeQueryReply frames a legacy OP_REPLY with one document.
func writeQueryReply(w io.Writer, responseTo int32, doc []byte) error {
	body := make([]byte, 20+len(doc))
	binary.LittleEndian.PutUint32(body[0:4], 0)   // response flags
	binary.LittleEndian.PutUint64(body[4:12], 0)  // cursor id
	binary.LittleEndian.PutUint32(body[12:16], 0) // starting from
	binary.LittleEndian.PutUint32(body[16:20], 1) // documents returned
	copy(body[20:], doc)
	header := make([]byte, 16)
	binary.LittleEndian.PutUint32(header[0:4], uint32(16+len(body)))
	binary.LittleEndian.PutUint32(header[4:8], 1)
	binary.LittleEndian.PutUint32(header[8:12], uint32(responseTo))
	binary.LittleEndian.PutUint32(header[12:16], uint32(1)) // OP_REPLY
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

// writeMessage frames one OP_MSG reply with the request's id as responseTo.
func writeMessage(w io.Writer, responseTo int32, doc []byte) error {
	body := make([]byte, 5+len(doc))
	body[4] = 0 // section kind 0; flags stay zero
	copy(body[5:], doc)
	header := make([]byte, 16)
	binary.LittleEndian.PutUint32(header[0:4], uint32(16+len(body)))
	binary.LittleEndian.PutUint32(header[4:8], 1)                   // response id
	binary.LittleEndian.PutUint32(header[8:12], uint32(responseTo)) // responseTo
	binary.LittleEndian.PutUint32(header[12:16], uint32(opMsg))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}
