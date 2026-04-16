package parser

import (
	"fmt"
	"os"
	"path/filepath"
)

// defaultPaths are tried in order when no explicit path is configured.
var defaultPaths = []string{
	"/etc/mediamtx/mediamtx.yml",
	"/etc/mediamtx.yml",
	"/var/lib/mediamtx/mediamtx.yml",
	"/opt/mediamtx/mediamtx.yml",
	"./mediamtx.yml",
}

// ParseResult holds the raw YAML and any metadata extracted from the mediamtx config file.
type ParseResult struct {
	// ResolvedPath is the file that was actually read.
	ResolvedPath string
	// RawYAML is the full file content, for display in the UI.
	RawYAML string
	// Available signals whether a config file was found at all.
	Available bool
}

// Parse attempts to read the mediamtx configuration file.
// If configPath is non-empty it is used directly; otherwise standard locations are tried.
// Errors are soft: if the file is not found or unreadable, Available is false.
func Parse(configPath string) *ParseResult {
	if configPath != "" {
		return readFile(configPath)
	}
	for _, p := range defaultPaths {
		r := readFile(p)
		if r.Available {
			return r
		}
	}
	return &ParseResult{}
}

func readFile(p string) *ParseResult {
	abs, err := filepath.Abs(p)
	if err != nil {
		return &ParseResult{}
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return &ParseResult{}
		}
		return &ParseResult{ResolvedPath: abs}
	}
	if len(data) > 512*1024 {
		return &ParseResult{
			ResolvedPath: abs,
			Available:    false,
			RawYAML:      fmt.Sprintf("# File too large to display (%d bytes)", len(data)),
		}
	}
	return &ParseResult{
		ResolvedPath: abs,
		RawYAML:      string(data),
		Available:    true,
	}
}
