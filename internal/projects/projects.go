// Package projects finds project folders and the git branch each is on.
package projects

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Project struct {
	Name   string
	Path   string
	Branch string // empty outside git
}

// Discover lists the extra folders first, in order, then every visible
// subfolder of each root, sorted by name. Missing folders are skipped, and a
// folder that shows up twice is listed once.
func Discover(roots, extra []string) []Project {
	var out []Project
	seen := map[string]bool{}
	add := func(path string) {
		path = filepath.Clean(path)
		if seen[path] {
			return
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			return
		}
		seen[path] = true
		out = append(out, Project{Name: filepath.Base(path), Path: path, Branch: Branch(path)})
	}

	for _, p := range extra {
		add(Expand(p))
	}
	for _, root := range roots {
		root = Expand(root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				names = append(names, e.Name())
			}
		}
		sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
		for _, n := range names {
			add(filepath.Join(root, n))
		}
	}
	return out
}

// Expand turns a leading ~ into the home folder.
func Expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// Branch reads the checked-out branch from .git/HEAD, without running git. A
// detached HEAD gives the short commit. Worktrees, whose .git is a file
// pointing at the real git folder, work too.
func Branch(path string) string {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		data, err := os.ReadFile(gitDir)
		if err != nil {
			return ""
		}
		dir, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir: ")
		if !ok {
			return ""
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(path, dir)
		}
		gitDir = dir
	}
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(string(head))
	if b, ok := strings.CutPrefix(ref, "ref: refs/heads/"); ok {
		return b
	}
	if len(ref) >= 7 {
		return ref[:7]
	}
	return ref
}
