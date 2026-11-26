package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	mkdir(t, root, "zeta")
	mkdir(t, root, "Alpha", ".git")
	writeFile(t, filepath.Join(root, "Alpha", ".git", "HEAD"), "ref: refs/heads/main\n")
	mkdir(t, root, ".hidden")
	writeFile(t, filepath.Join(root, "file.txt"), "")
	extra := mkdir(t, t.TempDir(), "nix-config")

	got := Discover([]string{root, filepath.Join(root, "missing")}, []string{extra, filepath.Join(root, "zeta")})
	var names []string
	for _, p := range got {
		names = append(names, p.Name)
	}
	want := []string{"nix-config", "zeta", "Alpha"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
	if got[2].Branch != "main" {
		t.Errorf("Alpha branch = %q", got[2].Branch)
	}
}

func TestBranch(t *testing.T) {
	dir := t.TempDir()
	if b := Branch(dir); b != "" {
		t.Errorf("no git: %q", b)
	}

	mkdir(t, dir, ".git")
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "0123456789abcdef\n")
	if b := Branch(dir); b != "0123456" {
		t.Errorf("detached: %q", b)
	}

	// A worktree: .git is a file pointing at the real git folder.
	wt := t.TempDir()
	real := mkdir(t, t.TempDir(), "worktrees", "wt")
	writeFile(t, filepath.Join(real, "HEAD"), "ref: refs/heads/feature/x\n")
	writeFile(t, filepath.Join(wt, ".git"), "gitdir: "+real+"\n")
	if b := Branch(wt); b != "feature/x" {
		t.Errorf("worktree: %q", b)
	}
}

func TestExpand(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := Expand("~/Projects"); got != filepath.Join(home, "Projects") {
		t.Errorf("got %q", got)
	}
	if got := Expand("/abs"); got != "/abs" {
		t.Errorf("got %q", got)
	}
}
