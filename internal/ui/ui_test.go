package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thiagokimo/nix-painel/internal/config"
)

func testConfig(t *testing.T) config.Config {
	cfg := config.Default()
	cfg.User, cfg.Host = "thiago", "t14"
	cfg.Tools = []config.Command{
		{Name: "nvim", Alias: "v", Cmd: []string{"nvim"}},
		{Name: "yazi", Alias: "y", Cmd: []string{"yazi"}},
		{Name: "btop", Alias: "top", Cmd: []string{"btop"}},
	}
	cfg.Scripts = []config.Command{
		{Name: "sysup", Alias: "su", Desc: "pull, rebuild, firmware", Cmd: []string{"sysup"}, Pause: true},
		{Name: "flakeup", Alias: "fu", Desc: "update inputs through a PR", Cmd: []string{"flakeup"}, Pause: true},
	}
	cfg.Projects.Roots = []string{t.TempDir()}
	cfg.Links = []config.LinkGroup{{Name: "dev", Links: []config.Link{
		{Name: "github", Alias: "gh", URL: "https://github.com"},
		{Name: "noogle", Alias: "ng", URL: "https://noogle.dev"},
	}}}
	return cfg
}

func names(m Model, idx []int) []string {
	var out []string
	for _, i := range idx {
		out = append(out, m.items[i].Name)
	}
	return out
}

func TestMatchPutsAliasFirst(t *testing.T) {
	m := New(testConfig(t))
	got := names(m, match(m.items, "y"))
	if len(got) == 0 || got[0] != "yazi" {
		t.Fatalf("alias y should rank yazi first: %v", got)
	}
	got = names(m, match(m.items, "su"))
	if got[0] != "sysup" {
		t.Fatalf("alias su: %v", got)
	}
	got = names(m, match(m.items, "flk"))
	if len(got) != 1 || got[0] != "flakeup" {
		t.Fatalf("fuzzy flk: %v", got)
	}
	if match(m.items, "  ") != nil {
		t.Fatal("blank query should match nothing")
	}
}

func TestAsURL(t *testing.T) {
	for in, want := range map[string]string{
		"github.com":       "https://github.com",
		"t14s:8096":        "",
		"pi.lan:8080/x":    "https://pi.lan:8080/x",
		"http://t14s:8096": "http://t14s:8096",
		"how to nix":       "",
		"nvim":             "",
	} {
		got, ok := asURL(in)
		if want == "" && ok || want != "" && got != want {
			t.Errorf("asURL(%q) = %q, %v", in, got, ok)
		}
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		next, _ := m.Update(key(k))
		m = next.(Model)
	}
	return m
}

func TestPromptAndNormalMode(t *testing.T) {
	m := New(testConfig(t))
	m = press(m, "f", "l", "a")
	if m.mode != modePrompt || len(m.matches) == 0 || m.items[m.matches[0]].Name != "flakeup" {
		t.Fatalf("typing should filter to flakeup: %v", names(m, m.matches))
	}
	m = press(m, "tab")
	if m.input.Value() != "flakeup" {
		t.Errorf("tab should complete, got %q", m.input.Value())
	}
	m = press(m, "esc")
	if m.input.Value() != "" || m.mode != modePrompt {
		t.Error("first esc clears the prompt")
	}
	m = press(m, "esc")
	if m.mode != modeNormal {
		t.Error("second esc goes to normal mode")
	}
	m = press(m, "j", "j")
	if p := m.panels[m.focus]; p.kind != Tool || p.cursor != 2 {
		t.Errorf("j should move down tools: %+v", p)
	}
	m = press(m, "l")
	if m.panels[m.focus].kind != Script {
		t.Error("l should move to the next panel")
	}
	m = press(m, "/")
	if m.mode != modePrompt {
		t.Error("/ goes back to the prompt")
	}
}

func TestEnterRunsCommand(t *testing.T) {
	m := New(testConfig(t))
	m = press(m, "s", "u")
	_, cmd := m.Update(key("enter"))
	if cmd == nil {
		t.Fatal("enter on a match should return a command")
	}
}

func TestShare(t *testing.T) {
	for _, tc := range []struct {
		wants []int
		total int
	}{
		{[]int{5, 4}, 30},
		{[]int{20, 20}, 30},
		{[]int{5, 40, 3}, 30},
		{[]int{2}, 10},
	} {
		got := share(tc.wants, tc.total)
		sum := 0
		for _, v := range got {
			sum += v
		}
		if sum != tc.total {
			t.Errorf("share(%v, %d) = %v, sums to %d", tc.wants, tc.total, got, sum)
		}
	}
	if got := share([]int{5, 40}, 30); got[0] != 5 {
		t.Errorf("a panel that fits should get what it needs: %v", got)
	}
}

func TestViewFitsTheTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 20}, {80, 24}, {140, 40}, {200, 12}} {
		m := New(testConfig(t))
		m.now = time.Date(2026, 11, 26, 14, 32, 0, 0, time.UTC)
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(Model)
		for _, keys := range [][]string{nil, {"g"}, {"esc", "esc", "j"}} {
			v := press(m, keys...).View()
			lines := strings.Split(v, "\n")
			if len(lines) != size[1] {
				t.Errorf("%v %v: %d lines, want %d", size, keys, len(lines), size[1])
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w > size[0] {
					t.Errorf("%v %v: line %d is %d wide: %q", size, keys, i, w, l)
				}
			}
		}
	}
}
