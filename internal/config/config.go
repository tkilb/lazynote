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
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/tkilb/lazynote/internal/store"
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

	// maxSourceRunes bounds one source name.
	maxSourceRunes = 40
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
	UI        UI
	// Sources are the sources Quick Add offers, the first being the default.
	Sources []string
	// Kinds are the kinds the config adds to the built-in ones, or whose
	// color, badge or meaning it changes. Empty means the built-ins as they
	// are.
	Kinds []store.KindSpec
	// Templates are the files new notes are made from.
	Templates Templates
	// Digest configures the agentic import.
	Digest Digest
}

// Templates names the template files. An empty path means no template.
type Templates struct {
	// Daily is the path of the template for a new daily note: absolute (a
	// leading ~ is expanded), or relative to the notebook root.
	Daily string
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

// Select returns the index of the notebook to open: the one at the path last
// used when it is still configured and available, else the first available
// one. It reports false when none is available.
func (c Config) Select(last string) (int, bool) {
	first, found := 0, false
	for i, nb := range c.Notebooks {
		if !nb.Available {
			continue
		}
		if last != "" && nb.Path == last {
			return i, true
		}
		if !found {
			first, found = i, true
		}
	}
	return first, found
}

// Retention holds the daily-note retention settings.
type Retention struct {
	// DailyDays is how many days an archived daily note is kept.
	DailyDays int
}

// UI holds the interface settings. The zero value is the default.
type UI struct {
	// MouseOff is set when ui.mouse is false in the config file.
	MouseOff bool
}

// MouseEnabled reports whether mouse support is on: true unless ui.mouse is
// explicitly false.
func (u UI) MouseEnabled() bool { return !u.MouseOff }

type rawConfig struct {
	Notebooks []string  `yaml:"notebooks"`
	Sources   yaml.Node `yaml:"sources"`
	Kinds     yaml.Node `yaml:"kinds"`
	Templates yaml.Node `yaml:"templates"`
	Digest    yaml.Node `yaml:"digest"`
	Retention struct {
		Daily string `yaml:"daily"`
	} `yaml:"retention"`
	UI struct {
		Mouse *bool `yaml:"mouse"`
	} `yaml:"ui"`
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

	sources, err := parseSources(&raw.Sources)
	if err != nil {
		return Config{}, err
	}

	kinds, err := parseKinds(&raw.Kinds)
	if err != nil {
		return Config{}, err
	}

	templates, err := parseTemplates(&raw.Templates, home)
	if err != nil {
		return Config{}, err
	}

	digest, err := parseDigest(&raw.Digest, home)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{Retention: Retention{DailyDays: days}, Sources: sources, Kinds: kinds, Templates: templates, Digest: digest}
	cfg.UI.MouseOff = raw.UI.Mouse != nil && !*raw.UI.Mouse
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

// parseTemplates reads the templates section: a mapping of template names to
// file paths. Only daily is known, and any other name is an error that names
// it. An absent or empty section means no templates.
func parseTemplates(n *yaml.Node, home string) (Templates, error) {
	var t Templates
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return t, nil
	}
	if n.Kind != yaml.MappingNode {
		return t, errors.New("templates must be a mapping, for example: templates: {daily: ~/Notes/work/inbox/templates/daily.md}")
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		name, val := n.Content[i].Value, n.Content[i+1]
		if name != "daily" {
			return t, fmt.Errorf("templates entry %q is not known; the only template is daily", name)
		}
		path := strings.TrimSpace(val.Value)
		if val.Kind != yaml.ScalarNode || path == "" {
			return t, errors.New("templates daily must be the path of a file, for example: ~/Notes/work/inbox/templates/daily.md")
		}
		t.Daily = expandHome(path, home)
	}
	return t, nil
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

// DefaultSources are the sources offered when the config lists none.
var DefaultSources = []string{"adhoc", "meeting", "conversation", "email", "web", "document"}

// parseSources reads the sources list. An absent, null or empty list gives a
// copy of DefaultSources. Entries are trimmed and must be non-empty, free of
// control characters, at most maxSourceRunes long and distinct ignoring case.
func parseSources(n *yaml.Node) ([]string, error) {
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return append([]string(nil), DefaultSources...), nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, errors.New("sources must be a list of names, for example: sources: [adhoc, meeting]")
	}
	if len(n.Content) == 0 {
		return append([]string(nil), DefaultSources...), nil
	}
	var out []string
	seen := map[string]bool{}
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode {
			return nil, errors.New("sources entries must be plain names")
		}
		name, err := ValidateSource(item.Value)
		if err != nil {
			return nil, fmt.Errorf("sources entry %q: %w", item.Value, err)
		}
		if seen[strings.ToLower(name)] {
			return nil, fmt.Errorf("sources entry %q is listed twice", name)
		}
		seen[strings.ToLower(name)] = true
		out = append(out, name)
	}
	return out, nil
}

// kindKeys are the keys an entry of the kinds list may have.
var kindKeys = map[string]bool{"name": true, "color": true, "meaning": true, "badge": true}

// parseKinds reads the kinds list: entries with a name and optionally a
// color, meaning and badge. A key that is not one of those is an error, so a
// typo is not silently ignored. Absent or empty means no changes.
func parseKinds(n *yaml.Node) ([]store.KindSpec, error) {
	if n.Kind == 0 || (n.Kind == yaml.ScalarNode && n.Tag == "!!null") {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, errors.New("kinds must be a list, for example: kinds: [{name: recipe, color: \"#ff8800\"}]")
	}
	specs := make([]store.KindSpec, 0, len(n.Content))
	for i, item := range n.Content {
		where := fmt.Sprintf("kinds entry %d", i+1)
		if item.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("%s must hold a name and optionally a color, meaning and badge", where)
		}
		var spec store.KindSpec
		for j := 0; j+1 < len(item.Content); j += 2 {
			key, val := item.Content[j], item.Content[j+1]
			if !kindKeys[key.Value] {
				return nil, fmt.Errorf("%s: unknown key %q (use name, color, meaning or badge)", where, key.Value)
			}
			if val.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("%s: %s must be text", where, key.Value)
			}
			switch key.Value {
			case "name":
				spec.Name = val.Value
			case "color":
				spec.Color = val.Value
			case "meaning":
				spec.Meaning = val.Value
			case "badge":
				spec.Badge = val.Value
			}
		}
		specs = append(specs, spec)
	}
	return store.ValidateKindSpecs(specs)
}

// ValidateSource trims name and checks that it can be a source: not blank, no
// control characters, at most maxSourceRunes long. Any other word is fine.
func ValidateSource(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("must not be blank")
	case strings.ContainsFunc(name, unicode.IsControl):
		return "", errors.New("must not contain control characters")
	case utf8.RuneCountInString(name) > maxSourceRunes:
		return "", fmt.Errorf("must be at most %d characters", maxSourceRunes)
	}
	return name, nil
}
