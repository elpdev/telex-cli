package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type UIPrefs struct {
	Theme          string `toml:"theme,omitempty"`
	SidebarVisible *bool  `toml:"sidebar_visible,omitempty"`
}

func PrefsPath() string { return filepath.Join(Dir(), "prefs.toml") }

func PrefsPathFor(override string) string {
	if override == "" {
		return PrefsPath()
	}
	return filepath.Join(filepath.Dir(filepath.Clean(override)), "prefs.toml")
}

func LoadPrefs(path string) (*UIPrefs, error) {
	var prefs UIPrefs
	if _, err := toml.DecodeFile(path, &prefs); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &UIPrefs{}, nil
		}
		return nil, err
	}
	return &prefs, nil
}

func (p *UIPrefs) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer closeSilently(f)
	return toml.NewEncoder(f).Encode(p)
}
