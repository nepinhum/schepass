package message

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nepinhum/schepass/internal/storage"
)

var (
	messages map[string]any
	mu       sync.RWMutex
)

func LoadMessages(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var tree map[string]any
	if err := json.Unmarshal(data, &tree); err != nil {
		return err
	}

	mu.Lock()
	messages = tree
	mu.Unlock()
	return nil
}

// LoadOrSeed loads messages from targetPath, seeding from defaultPath if needed.
func LoadOrSeed(targetPath, defaultPath string) error {
	if err := LoadMessages(targetPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := SeedFromDefault(defaultPath, targetPath); err != nil {
		return err
	}
	return LoadMessages(targetPath)
}

// SeedFromDefault copies the default messages into the target location.
func SeedFromDefault(defaultPath, targetPath string) error {
	data, err := os.ReadFile(defaultPath)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(targetPath, data, 0o600)
}

// Msg gets a string message by dot-path key.
// Example: Msg("test.test")
func Msg(key string, a ...any) string {
	mu.RLock()
	tree := messages
	mu.RUnlock()
	if tree == nil {
		return ""
	}

	v, ok := resolveKey(tree, key)
	if !ok {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	if len(a) > 0 {
		return fmt.Sprintf(s, a...)
	}
	return s
}

// Arr gets []string from an array key.
func Arr(key string) []string {
	mu.RLock()
	tree := messages
	mu.RUnlock()
	if tree == nil {
		return nil
	}

	v, ok := resolveKey(tree, key)
	if !ok {
		return nil
	}

	raw, ok := v.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(raw))
	for _, it := range raw {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// IsBranch checks if key points to a JSON object (map).
func IsBranch(key string) bool {
	mu.RLock()
	tree := messages
	mu.RUnlock()
	if tree == nil {
		return false
	}

	v, ok := resolveKey(tree, key)
	if !ok {
		return false
	}

	_, ok = v.(map[string]any)
	return ok
}

// HasKey checks if a key exists in messages.json
func HasKey(key string) bool {
	mu.RLock()
	tree := messages
	mu.RUnlock()
	if tree == nil {
		return false
	}

	_, ok := resolveKey(tree, key)
	return ok
}

func resolveKey(tree map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = tree

	for _, p := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[p]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
