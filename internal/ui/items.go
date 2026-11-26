package ui

import (
	"regexp"
	"strings"

	"github.com/sahilm/fuzzy"
	"github.com/thiagokimo/painel/internal/config"
	"github.com/thiagokimo/painel/internal/projects"
)

type Kind int

const (
	Tool Kind = iota
	Script
	Project
	Link
)

var kindTitles = [...]string{"tools", "scripts", "projects", "links"}

// Item is one entry on screen, of any kind.
type Item struct {
	Kind  Kind
	Name  string
	Alias string
	Desc  string

	Cmd   []string // tools and scripts
	Pause bool

	Path   string // projects
	Branch string

	URL   string // links
	Group string
}

func buildItems(cfg config.Config) []Item {
	var items []Item
	for _, c := range cfg.Tools {
		items = append(items, Item{Kind: Tool, Name: c.Name, Alias: c.Alias, Desc: c.Desc, Cmd: c.Cmd, Pause: c.Pause})
	}
	for _, c := range cfg.Scripts {
		items = append(items, Item{Kind: Script, Name: c.Name, Alias: c.Alias, Desc: c.Desc, Cmd: c.Cmd, Pause: c.Pause})
	}
	for _, p := range projects.Discover(cfg.Projects.Roots, cfg.Projects.Extra) {
		items = append(items, Item{Kind: Project, Name: p.Name, Path: p.Path, Branch: p.Branch})
	}
	for _, g := range cfg.Links {
		for _, l := range g.Links {
			items = append(items, Item{Kind: Link, Name: l.Name, Alias: l.Alias, URL: l.URL, Group: g.Name})
		}
	}
	return items
}

// match ranks items for the prompt: an exact alias first, then an exact name,
// then fuzzy matches on the name, best first.
func match(items []Item, query string) []int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []int
	seen := map[int]bool{}
	add := func(i int) {
		if !seen[i] {
			seen[i] = true
			out = append(out, i)
		}
	}
	for i, it := range items {
		if it.Alias != "" && strings.ToLower(it.Alias) == q {
			add(i)
		}
	}
	for i, it := range items {
		if strings.ToLower(it.Name) == q {
			add(i)
		}
	}
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.Name
	}
	for _, m := range fuzzy.Find(q, names) {
		add(m.Index)
	}
	return out
}

var (
	hasScheme  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)
	looksHosty = regexp.MustCompile(`^[\w-]+(\.[\w-]+)+(:\d+)?(/\S*)?$`)
)

// asURL turns prompt text that looks like an address into a URL.
func asURL(q string) (string, bool) {
	q = strings.TrimSpace(q)
	switch {
	case hasScheme.MatchString(q):
		return q, true
	case looksHosty.MatchString(q):
		return "https://" + q, true
	}
	return "", false
}
