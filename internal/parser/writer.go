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

// UpsertPath adds or replaces a paths.<name> entry in mediamtx.yml.
// If configPath is empty, the call is a no-op (persistence disabled).
// Comments and unrelated sections are preserved.
func UpsertPath(configPath, name string, cfg mediamtx.PathConfig) error {
	if configPath == "" {
		return nil
	}
	return mutatePathsFile(configPath, func(paths *yaml.Node) {
		entry := pathEntryNode(cfg)
		for i := 0; i < len(paths.Content); i += 2 {
			if paths.Content[i].Value == name {
				paths.Content[i+1] = entry
				return
			}
		}
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

func pathEntryNode(cfg mediamtx.PathConfig) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode}
	addStr := func(k, v string) {
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: v},
		)
	}
	addBool := func(k string, v bool) {
		val := "false"
		if v {
			val = "true"
		}
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val},
		)
	}
	if cfg.Source != "" {
		addStr("source", cfg.Source)
	}
	if cfg.SourceOnDemand {
		addBool("sourceOnDemand", true)
	}
	if cfg.Record {
		addBool("record", true)
	}
	if cfg.MaxReaders > 0 {
		addStr("maxReaders", strconv.Itoa(cfg.MaxReaders))
	}
	return n
}
