package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"mediamtx-ui/internal/mediamtx"
)

const sampleYML = `# top comment
api: true
authMethod: http
paths:
  cam1:
    source: rtsp://10.0.0.1:554/profile1
    rtspTransport: tcp
    sourceOnDemand: false
  secure:
    source: publisher
    readUser: viewer
    readPass: s3cret
`

func writeSample(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mediamtx.yml")
	if err := os.WriteFile(p, []byte(sampleYML), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readPaths(t *testing.T, p string) map[string]map[string]any {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Paths
}

func TestUpsertPathPreservesUnmanagedKeys(t *testing.T) {
	p := writeSample(t)

	// Mimics "sync config to yml": the API returns the path without the
	// readPass the file holds (or with it redacted, which we do not model).
	if err := UpsertPath(p, "secure", mediamtx.PathConfig{Source: "publisher", MaxReaders: 5}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertPath(p, "cam1", mediamtx.PathConfig{Source: "rtsp://10.0.0.2:554/profile1", SourceOnDemand: true}); err != nil {
		t.Fatal(err)
	}

	paths := readPaths(t, p)
	sec := paths["secure"]
	if sec["readUser"] != "viewer" || sec["readPass"] != "s3cret" {
		t.Fatalf("credentials lost: %#v", sec)
	}
	if sec["maxReaders"] != 5 {
		t.Fatalf("maxReaders not written as int: %#v", sec["maxReaders"])
	}
	cam := paths["cam1"]
	if cam["rtspTransport"] != "tcp" {
		t.Fatalf("rtspTransport lost: %#v", cam)
	}
	if cam["source"] != "rtsp://10.0.0.2:554/profile1" || cam["sourceOnDemand"] != true {
		t.Fatalf("managed keys not updated: %#v", cam)
	}

	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "# top comment") {
		t.Fatal("comment lost")
	}
}

func TestUpsertPathClearsZeroManagedKeys(t *testing.T) {
	p := writeSample(t)
	if err := UpsertPath(p, "cam1", mediamtx.PathConfig{Source: "rtsp://10.0.0.1:554/profile1"}); err != nil {
		t.Fatal(err)
	}
	cam := readPaths(t, p)["cam1"]
	if _, ok := cam["sourceOnDemand"]; ok {
		t.Fatalf("sourceOnDemand=false should be dropped (default): %#v", cam)
	}
	if cam["rtspTransport"] != "tcp" {
		t.Fatalf("rtspTransport lost: %#v", cam)
	}
}

func TestUpsertPathAddsNew(t *testing.T) {
	p := writeSample(t)
	if err := UpsertPath(p, "new", mediamtx.PathConfig{Source: "publisher", Record: true}); err != nil {
		t.Fatal(err)
	}
	n := readPaths(t, p)["new"]
	if n["source"] != "publisher" || n["record"] != true {
		t.Fatalf("new path wrong: %#v", n)
	}
}

func TestUpsertPathRefusesRedacted(t *testing.T) {
	p := writeSample(t)
	before, _ := os.ReadFile(p)
	if err := UpsertPath(p, "cam1", mediamtx.PathConfig{Source: RedactedCredential}); err == nil {
		t.Fatal("expected error for redacted value")
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Fatal("file modified despite refusal")
	}
}
