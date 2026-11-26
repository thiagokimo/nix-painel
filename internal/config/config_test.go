package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissingFileGivesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tools) == 0 || cfg.Theme.Accent == "" || len(cfg.Browser) == 0 {
		t.Fatalf("defaults missing: %+v", cfg)
	}
}

func TestFileOverridesDefaults(t *testing.T) {
	cfg, err := Load(write(t, `{
		"host": "t14",
		"theme": {"accent": "#123456"},
		"tools": [{"name": "kitty", "cmd": ["kitty"]}],
		"scripts": [{"name": "sysup", "alias": "su", "cmd": ["sysup"], "pause": true}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "t14" || cfg.Theme.Accent != "#123456" {
		t.Errorf("host/accent not read: %q %q", cfg.Host, cfg.Theme.Accent)
	}
	if cfg.Theme.Green != defaultTheme.Green {
		t.Errorf("a color left out should keep its default, got %q", cfg.Theme.Green)
	}
	if len(cfg.Tools) != 1 || cfg.Tools[0].Name != "kitty" {
		t.Errorf("tools should replace the defaults: %+v", cfg.Tools)
	}
	if !cfg.Scripts[0].Pause {
		t.Error("pause not read")
	}
	if len(cfg.Projects.Open) == 0 {
		t.Error("projects.open left out should keep its default")
	}
}

func TestRejectsBadConfig(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":    `{"tool": []}`,
		"no cmd":           `{"tools": [{"name": "x"}]}`,
		"duplicate alias":  `{"tools": [{"name": "a", "alias": "x", "cmd": ["a"]}], "links": [{"name": "g", "links": [{"name": "b", "alias": "x", "url": "u"}]}]}`,
		"link without url": `{"links": [{"name": "g", "links": [{"name": "b"}]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, body)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestErrorNamesTheFile(t *testing.T) {
	p := write(t, `{`)
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("error should name %s: %v", p, err)
	}
}
