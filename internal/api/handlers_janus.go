package api

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// janusSession represents an active viewer connected via the Janus proxy.
type janusSession struct {
	Username    string    `json:"username"`
	ConnectedAt time.Time `json:"connectedAt"`
}

// janusRegistry tracks active Janus WebSocket proxy sessions in memory.
type janusRegistry struct {
	mu       sync.RWMutex
	sessions map[string]janusSession
}

func newJanusRegistry() *janusRegistry {
	return &janusRegistry{sessions: make(map[string]janusSession)}
}

func (reg *janusRegistry) add(id, username string) {
	reg.mu.Lock()
	reg.sessions[id] = janusSession{Username: username, ConnectedAt: time.Now()}
	reg.mu.Unlock()
}

func (reg *janusRegistry) remove(id string) {
	reg.mu.Lock()
	delete(reg.sessions, id)
	reg.mu.Unlock()
}

func (reg *janusRegistry) list() []janusSession {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	out := make([]janusSession, 0, len(reg.sessions))
	for _, s := range reg.sessions {
		out = append(out, s)
	}
	return out
}

// handleJanusWS proxies a WebSocket upgrade to the internal Janus backend,
// tracking the authenticated user for the duration of the connection.
func (s *Server) handleJanusWS(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	username := ""
	if claims != nil {
		username = claims.Username
	}

	backConn, err := net.DialTimeout("tcp", s.cfg.Janus.WSAddr, 5*time.Second)
	if err != nil {
		http.Error(w, "janus unavailable", http.StatusBadGateway)
		return
	}
	defer backConn.Close()

	// Forward the WebSocket upgrade request to Janus verbatim.
	var req bytes.Buffer
	fmt.Fprintf(&req, "GET /janus HTTP/1.1\r\nHost: %s\r\n", s.cfg.Janus.WSAddr)
	for k, vv := range r.Header {
		for _, v := range vv {
			fmt.Fprintf(&req, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(&req, "\r\n")
	if _, err := backConn.Write(req.Bytes()); err != nil {
		http.Error(w, "janus write failed", http.StatusBadGateway)
		return
	}

	// Read the response status line + headers as raw bytes (no http.ReadResponse —
	// it canonicalizes header casing and resp.Write tries to consume a 101 body,
	// both of which break the WebSocket upgrade).
	br := bufio.NewReader(backConn)
	var respHead bytes.Buffer
	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			http.Error(w, "janus read failed", http.StatusBadGateway)
			return
		}
		respHead.Write(line)
		if bytes.Equal(line, []byte("\r\n")) {
			break
		}
	}
	if !bytes.HasPrefix(respHead.Bytes(), []byte("HTTP/1.1 101")) {
		http.Error(w, "janus did not upgrade", http.StatusBadGateway)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	// Forward the 101 response verbatim.
	if _, err := clientConn.Write(respHead.Bytes()); err != nil {
		return
	}

	// Track this session.
	sid := fmt.Sprintf("%p", clientConn)
	s.janusReg.add(sid, username)
	defer s.janusReg.remove(sid)

	// Bidirectional bridge: browser ↔ Janus.
	done := make(chan struct{}, 2)
	go func() { io.Copy(backConn, clientBuf); done <- struct{}{} }()
	go func() { io.Copy(clientConn, br); done <- struct{}{} }() // br drains any buffered frames
	<-done
}

// handleJanusViewers returns the list of users currently watching via Janus.
func (s *Server) handleJanusViewers(w http.ResponseWriter, _ *http.Request) {
	jsonOK(w, s.janusReg.list())
}
