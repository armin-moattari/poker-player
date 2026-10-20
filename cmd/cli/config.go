package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/armin-moattari/poker-player/internal/advisor"
)

// configPath returns where the advisor settings file lives:
// <user config dir>/poker-player/advisor.json, for example
// ~/Library/Application Support/poker-player/advisor.json on macOS or
// ~/.config/poker-player/advisor.json on Linux.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "poker-player", "advisor.json"), nil
}

// loadConfig reads the settings file at path on top of the defaults, so the
// file only needs the settings you want to change. A missing file is not an
// error: you get the defaults. Unknown keys are rejected, so a typo like
// "bet_fracton" is reported instead of silently ignored.
func loadConfig(path string) (advisor.Config, error) {
	cfg := advisor.DefaultConfig()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("in %s: %w", path, err)
	}
	return cfg, nil
}

// writeConfig saves cfg as indented JSON at path, creating the directory if
// needed. It refuses to overwrite an existing file, so running "config
// --init" twice can't wipe out your edits.
func writeConfig(path string, cfg advisor.Config) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists; edit it instead", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
