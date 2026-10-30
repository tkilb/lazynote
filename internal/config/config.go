// Package config loads lazynote's user-editable YAML config from
// ~/.config/lazynote/config.yaml (the same fixed path on every OS, as in
// lazytask). It lists the notebooks and the daily-note retention period.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	dirName  = "lazynote"
	fileName = "config.yaml"

	// DefaultRetentionDays is used when retention.daily is absent.
	DefaultRetentionDays = 30

	retentionSuffix = "d"

	// maxRetentionDays caps retention.daily (100 years) so later date
	// arithmetic cannot overflow.
	maxRetentionDays = 36500
)

// ExampleConfig is shown when no notebooks are configured.
const ExampleConfig = `notebooks:
  - ~/Notes/work
  - ~/Notes/personal
retention:
  daily: 30d
`

// ErrNoNotebooks reports an absent file or an empty notebooks list.
var ErrNoNotebooks = errors.New("no notebooks configured; add a notebooks list, for example:\n\n" + ExampleConfig)

// Config is the parsed, validated configuration.
type Config struct {
	// Notebooks keeps the configured order; the first entry is the default.
	Notebooks []Notebook
	Retention Retention
}

// Notebook is one configured notebook root.
type Notebook struct {
	// Path is the absolute root with "~" expanded.
	Path string
	// Name is the directory's base name, shown in the Notebook panel.
	Name string
	// Available is false when Path is not an existing directory. Such a
	// notebook is listed but cannot be selected.
	Available bool
}

// Retention holds the daily-note retention settings.
type Retention struct {
	// DailyDays is how many days an archived daily note is kept.
	DailyDays int
}

type rawConfig struct {
	Notebooks []string `yaml:"notebooks"`
	Retention struct {
		Daily string `yaml:"daily"`
	} `yaml:"retention"`
}

// DefaultPath returns ~/.config/lazynote/config.yaml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", dirName, fileName), nil
}

// Load reads and parses the config file at path. A missing file is reported
// as ErrNoNotebooks.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("config file %s not found: %w", path, ErrNoNotebooks)
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("find home directory: %w", err)
	}
	return Parse(data, home)
}

// Parse validates data and expands "~" using home.
func Parse(data []byte, home string) (Config, error) {
	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if len(raw.Notebooks) == 0 {
		return Config{}, ErrNoNotebooks
	}

	days, err := parseRetention(raw.Retention.Daily)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{Retention: Retention{DailyDays: days}}
	for _, p := range raw.Notebooks {
		nb, err := newNotebook(p, home)
		if err != nil {
			return Config{}, err
		}
		cfg.Notebooks = append(cfg.Notebooks, nb)
	}
	return cfg, nil
}

func newNotebook(p, home string) (Notebook, error) {
	expanded := expandHome(strings.TrimSpace(p), home)
	if expanded == "" || !filepath.IsAbs(expanded) {
		return Notebook{}, fmt.Errorf("notebook path %q must be absolute or start with ~", p)
	}
	clean := filepath.Clean(expanded)
	info, err := os.Stat(clean)
	return Notebook{
		Path:      clean,
		Name:      filepath.Base(clean),
		Available: err == nil && info.IsDir(),
	}, nil
}

// expandHome replaces a leading "~" or "~/" with home.
func expandHome(p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	}
	return p
}

// parseRetention reads "<positive whole number>d". Empty means the default.
func parseRetention(s string) (int, error) {
	if s == "" {
		return DefaultRetentionDays, nil
	}
	digits, ok := strings.CutSuffix(s, retentionSuffix)
	if !ok || digits == "" {
		return 0, fmt.Errorf("retention.daily %q must be a positive whole number followed by d, for example 30d", s)
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("retention.daily %q must be a positive whole number followed by d, for example 30d", s)
		}
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 || n > maxRetentionDays {
		return 0, fmt.Errorf("retention.daily %q must be a positive whole number followed by d, for example 30d", s)
	}
	return n, nil
}
