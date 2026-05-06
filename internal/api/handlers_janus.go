package api

import (
	"bufio"
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
	fmt.Fprintf(backConn, "GET /janus HTTP/1.1\r\nHost: %s\r\n", s.cfg.Janus.WSAddr)
	for k, vv := range r.Header {
		for _, v := range vv {
			fmt.Fprintf(backConn, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(backConn, "\r\n")

	br := bufio.NewReader(backConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil || resp.StatusCode != http.StatusSwitchingProtocols {
		http.Error(w, "janus handshake failed", http.StatusBadGateway)
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

	// Send the 101 Switching Protocols response to the browser.
	if err := resp.Write(clientConn); err != nil {
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
