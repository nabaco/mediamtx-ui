package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"

	"mediamtx-ui/internal/mediamtx"
)

// writeMu serializes concurrent writes to the config file.
var writeMu sync.Mutex

// RedactedCredential is the placeholder mediamtx >= 1.20.1 returns from the
// API in place of credentials (readPass, publishPass, authInternalUsers pass).
// It must never be written back to mediamtx.yml.
const RedactedCredential = "<redacted>"

// UpsertPath adds or updates a paths.<name> entry in mediamtx.yml.
// If configPath is empty, the call is a no-op (persistence disabled).
//
// Only the keys the UI manages (source, sourceOnDemand, record, maxReaders)
// are touched; every other key already in the entry (rtspTransport,
// readUser/readPass, runOnInit, ...) is preserved as-is. That matters because
// the API no longer returns real credentials, so the file is the only place
// they survive. Comments and unrelated sections are preserved.
func UpsertPath(configPath, name string, cfg mediamtx.PathConfig) error {
	if configPath == "" {
		return nil
	}
	if cfg.Source == RedactedCredential {
		return fmt.Errorf("refusing to write redacted value for paths.%s.source", name)
	}
	return mutatePathsFile(configPath, func(paths *yaml.Node) {
		for i := 0; i < len(paths.Content); i += 2 {
			if paths.Content[i].Value == name {
				entry := paths.Content[i+1]
				if entry.Kind != yaml.MappingNode {
					entry = &yaml.Node{Kind: yaml.MappingNode}
					paths.Content[i+1] = entry
				}
				applyManagedKeys(entry, cfg)
				return
			}
		}
		entry := &yaml.Node{Kind: yaml.MappingNode}
		applyManagedKeys(entry, cfg)
		paths.Content = append(paths.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: name},
			entry,
		)
	})
}

// DeletePath removes a paths.<name> entry. No-op if absent or configPath empty.
func DeletePath(configPath, name string) error {
	if configPath == "" {
		return nil
	}
	return mutatePathsFile(configPath, func(paths *yaml.Node) {
		for i := 0; i < len(paths.Content); i += 2 {
			if paths.Content[i].Value == name {
				paths.Content = append(paths.Content[:i], paths.Content[i+2:]...)
				return
			}
		}
	})
}

func mutatePathsFile(configPath string, mutate func(paths *yaml.Node)) error {
	writeMu.Lock()
	defer writeMu.Unlock()

	abs, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("read %s: %w", abs, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse %s: %w", abs, err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("unexpected yaml structure in %s", abs)
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return fmt.Errorf("root of %s is not a mapping", abs)
	}

	paths := findOrCreateMapping(doc, "paths")
	mutate(paths)

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, abs)
}

func findOrCreateMapping(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			v := parent.Content[i+1]
			if v.Kind != yaml.MappingNode {
				v.Kind = yaml.MappingNode
				v.Tag = ""
				v.Value = ""
				v.Content = nil
			}
			return v
		}
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	v := &yaml.Node{Kind: yaml.MappingNode}
	parent.Content = append(parent.Content, k, v)
	return v
}

// applyManagedKeys sets (or removes, when zero-valued) the UI-managed keys in
// a path mapping node, leaving all other keys untouched.
func applyManagedKeys(n *yaml.Node, cfg mediamtx.PathConfig) {
	set := func(k, v, tag string) {
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == k {
				n.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: tag}
				return
			}
		}
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: tag},
		)
	}
	del := func(k string) {
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == k {
				n.Content = append(n.Content[:i], n.Content[i+2:]...)
				return
			}
		}
	}
	if cfg.Source != "" {
		set("source", cfg.Source, "")
	} else {
		del("source")
	}
	if cfg.SourceOnDemand {
		set("sourceOnDemand", "true", "!!bool")
	} else {
		del("sourceOnDemand")
	}
	if cfg.Record {
		set("record", "true", "!!bool")
	} else {
		del("record")
	}
	if cfg.MaxReaders > 0 {
		set("maxReaders", strconv.Itoa(cfg.MaxReaders), "!!int")
	} else {
		del("maxReaders")
	}
}
