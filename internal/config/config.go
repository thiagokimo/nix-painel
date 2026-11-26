// Package config loads painel's configuration: the tools, scripts, project
// folders and links it shows, and the colors it draws them in.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Command is something painel runs in the terminal: a tool such as nvim, or a
// script such as sysup.
type Command struct {
	Name  string   `json:"name"`
	Alias string   `json:"alias,omitempty"`
	Desc  string   `json:"desc,omitempty"`
	Cmd   []string `json:"cmd"`
	// Pause waits for enter after the command exits, so its output stays on
	// screen instead of vanishing when painel comes back.
	Pause bool `json:"pause,omitempty"`
}

// Opener is one way to open a project, such as nvim or Claude Code. Key picks
// it when painel asks; it defaults to the first letter of the name.
type Opener struct {
	Name string   `json:"name"`
	Key  string   `json:"key,omitempty"`
	Cmd  []string `json:"cmd"`
}

// KeyOrDefault is Key, or the name's first letter.
func (o Opener) KeyOrDefault() string {
	if o.Key != "" {
		return o.Key
	}
	if o.Name == "" {
		return ""
	}
	return strings.ToLower(o.Name[:1])
}

// Projects says where to find project folders and what opens them.
type Projects struct {
	// Roots are folders whose subfolders are projects, such as ~/Projects.
	Roots []string `json:"roots"`
	// Extra are single project folders outside the roots, listed first.
	Extra []string `json:"extra"`
	// OpenWith runs inside the project on enter. With more than one,
	// painel asks which. Alt runs on y.
	OpenWith []Opener `json:"open_with"`
	Alt      []string `json:"alt"`
}

type Link struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
	URL   string `json:"url"`
}

type LinkGroup struct {
	Name  string `json:"name"`
	Links []Link `json:"links"`
}

// Theme holds colors as #rrggbb. Painel leaves the background alone, so the
// terminal's own (and its transparency) shows through.
type Theme struct {
	Fg        string `json:"fg"`
	FgBright  string `json:"fg_bright"`
	Muted     string `json:"muted"`
	Border    string `json:"border"`
	Selection string `json:"selection"`
	Accent    string `json:"accent"`
	Red       string `json:"red"`
	Orange    string `json:"orange"`
	Yellow    string `json:"yellow"`
	Green     string `json:"green"`
	Aqua      string `json:"aqua"`
	Blue      string `json:"blue"`
}

type Config struct {
	User     string      `json:"user"`
	Host     string      `json:"host"`
	Theme    Theme       `json:"theme"`
	Tools    []Command   `json:"tools"`
	Scripts  []Command   `json:"scripts"`
	Projects Projects    `json:"projects"`
	Links    []LinkGroup `json:"links"`
	// Browser opens links: the URL is added as its last argument.
	Browser []string `json:"browser"`
	// Search is a URL prefix for prompt text that matches nothing, such as
	// https://www.google.com/search?q=. Empty turns searching off.
	Search string `json:"search"`
}

// DefaultPath is $XDG_CONFIG_HOME/painel/config.json.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(dir, "painel", "config.json")
}

// Gruvbox dark, the same palette as the rest of the desktop.
var defaultTheme = Theme{
	Fg:        "#d5c4a1",
	FgBright:  "#fbf1c7",
	Muted:     "#928374",
	Border:    "#504945",
	Selection: "#3c3836",
	Accent:    "#8ec07c",
	Red:       "#fb4934",
	Orange:    "#fe8019",
	Yellow:    "#fabd2f",
	Green:     "#b8bb26",
	Aqua:      "#8ec07c",
	Blue:      "#83a598",
}

// Default is what painel shows without a config file.
func Default() Config {
	cfg := Config{
		Theme: defaultTheme,
		Tools: []Command{
			{Name: "nvim", Alias: "v", Desc: "editor", Cmd: []string{"nvim"}},
			{Name: "yazi", Alias: "y", Desc: "files", Cmd: []string{"yazi"}},
			{Name: "btop", Alias: "top", Desc: "processes", Cmd: []string{"btop"}},
		},
		Projects: Projects{
			Roots:    []string{"~/Projects"},
			OpenWith: []Opener{{Name: "nvim", Key: "v", Cmd: []string{"nvim", "."}}},
			Alt:      []string{"yazi"},
		},
		Browser: []string{"xdg-open"},
		Search:  "https://www.google.com/search?q=",
	}
	if u, err := user.Current(); err == nil {
		cfg.User = u.Username
	}
	if h, err := os.Hostname(); err == nil {
		cfg.Host = h
	}
	return cfg
}

// file is the config file as written: a nil field was left out.
type file struct {
	User     *string    `json:"user"`
	Host     *string    `json:"host"`
	Theme    *Theme     `json:"theme"`
	Tools    *[]Command `json:"tools"`
	Scripts  *[]Command `json:"scripts"`
	Projects *struct {
		Roots    *[]string `json:"roots"`
		Extra    *[]string `json:"extra"`
		OpenWith *[]Opener `json:"open_with"`
		Alt      *[]string `json:"alt"`
	} `json:"projects"`
	Links   *[]LinkGroup `json:"links"`
	Browser *[]string    `json:"browser"`
	Search  *string      `json:"search"`
}

func set[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// Load reads the config at path over the defaults: a list in the file replaces
// the default list, and a theme color left out keeps its default. A missing
// file gives the defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}

	// Theme colors decode over the defaults; everything else is replaced
	// whole, so a list in the file never mixes with the default one.
	f := file{Theme: &cfg.Theme}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	set(&cfg.User, f.User)
	set(&cfg.Host, f.Host)
	set(&cfg.Theme, f.Theme)
	set(&cfg.Tools, f.Tools)
	set(&cfg.Scripts, f.Scripts)
	if p := f.Projects; p != nil {
		set(&cfg.Projects.Roots, p.Roots)
		set(&cfg.Projects.Extra, p.Extra)
		set(&cfg.Projects.OpenWith, p.OpenWith)
		set(&cfg.Projects.Alt, p.Alt)
	}
	set(&cfg.Links, f.Links)
	set(&cfg.Browser, f.Browser)
	set(&cfg.Search, f.Search)

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	var problems []string
	aliases := map[string]string{}
	alias := func(a, owner string) {
		if a == "" {
			return
		}
		if prev, ok := aliases[a]; ok {
			problems = append(problems, fmt.Sprintf("alias %q is used by both %s and %s", a, prev, owner))
		}
		aliases[a] = owner
	}

	for _, list := range [][]Command{c.Tools, c.Scripts} {
		for _, cmd := range list {
			if cmd.Name == "" {
				problems = append(problems, "a command has no name")
			}
			if len(cmd.Cmd) == 0 {
				problems = append(problems, fmt.Sprintf("%s has no cmd", cmd.Name))
			}
			alias(cmd.Alias, cmd.Name)
		}
	}
	for _, g := range c.Links {
		for _, l := range g.Links {
			if l.Name == "" || l.URL == "" {
				problems = append(problems, fmt.Sprintf("a link in %s needs a name and a url", g.Name))
			}
			alias(l.Alias, l.Name)
		}
	}
	if len(c.Projects.OpenWith) == 0 {
		problems = append(problems, "projects.open_with is empty")
	}
	keys := map[string]string{}
	for _, o := range c.Projects.OpenWith {
		if o.Name == "" || len(o.Cmd) == 0 {
			problems = append(problems, "each projects.open_with entry needs a name and a cmd")
			continue
		}
		k := o.KeyOrDefault()
		if prev, ok := keys[k]; ok {
			problems = append(problems, fmt.Sprintf("projects.open_with key %q is used by both %s and %s", k, prev, o.Name))
		}
		keys[k] = o.Name
	}
	if len(c.Browser) == 0 {
		problems = append(problems, "browser is empty")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
