package api

import (
	"testing"

	"mediamtx-ui/internal/config"
)

func newAuthTestServer(apiUser, apiPass string) *Server {
	cfg := &config.Config{}
	cfg.MediaMTX.APIUser = apiUser
	cfg.MediaMTX.APIPass = apiPass
	return &Server{cfg: cfg}
}

func TestServiceActionsRequireAPICredentials(t *testing.T) {
	s := newAuthTestServer("mediamtx", "apipass")

	cases := []struct {
		name string
		req  mediamtxAuthRequest
		want bool
	}{
		{"api ok", mediamtxAuthRequest{Action: "api", User: "mediamtx", Password: "apipass"}, true},
		{"metrics ok", mediamtxAuthRequest{Action: "metrics", User: "mediamtx", Password: "apipass"}, true},
		{"pprof ok", mediamtxAuthRequest{Action: "pprof", User: "mediamtx", Password: "apipass"}, true},
		{"api wrong pass", mediamtxAuthRequest{Action: "api", User: "mediamtx", Password: "nope"}, false},
		{"api other user", mediamtxAuthRequest{Action: "api", User: "admin", Password: "apipass"}, false},
		{"api anonymous", mediamtxAuthRequest{Action: "api"}, false},
		{"unknown action", mediamtxAuthRequest{Action: "frobnicate", User: "mediamtx", Password: "apipass"}, false},
	}
	for _, c := range cases {
		got, _ := s.checkMediaMTXAuth(c.req)
		if got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestServiceActionsDeniedWithoutConfiguredCredentials(t *testing.T) {
	s := newAuthTestServer("", "")
	if ok, _ := s.checkMediaMTXAuth(mediamtxAuthRequest{Action: "api", User: "", Password: ""}); ok {
		t.Fatal("empty configured creds must not allow api")
	}
	s = newAuthTestServer("mediamtx", "")
	if ok, _ := s.checkMediaMTXAuth(mediamtxAuthRequest{Action: "api", User: "mediamtx", Password: ""}); ok {
		t.Fatal("empty configured pass must not allow api")
	}
}
