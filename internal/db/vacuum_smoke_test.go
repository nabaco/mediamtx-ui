package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneAndVacuum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	for i := 0; i < 20000; i++ {
		if err := WriteAuditLog(d, AuditEntry{Username: "u", StreamPath: "p", Action: "read", Protocol: "rtsp", RemoteAddr: "1.2.3.4", Allowed: true}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Exec("UPDATE audit_log SET created_at = datetime('now','-30 days') WHERE id % 2 = 0"); err != nil {
		t.Fatal(err)
	}

	n, err := PruneAuditLog(d, 14)
	if err != nil {
		t.Fatal(err)
	}
	if n != 10000 {
		t.Fatalf("pruned %d rows, want 10000", n)
	}

	before := size(t, path)
	if err := VacuumDB(d); err != nil {
		t.Fatal(err)
	}
	after := size(t, path)
	if after >= before {
		t.Fatalf("vacuum did not shrink file: %d -> %d", before, after)
	}

	var left int
	if err := d.QueryRow("SELECT count(*) FROM audit_log").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 10000 {
		t.Fatalf("%d rows left, want 10000", left)
	}
}

func size(t *testing.T, path string) int64 {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return st.Size()
}
