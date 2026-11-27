package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// A dialog floats over the screen: a rounded box as big as what it holds, its
// title (and a summary at the end) on the top edge and its keys on the bottom
// edge, in the middle of the screen, which goes dim behind it.
//
//	╭─ open nix-config ────  main ─╮
//	│ ❯ [v] nvim   nvim .          │
//	│   [c] claude claude          │
//	╰─ v/c pick · enter · esc ─────╯

type dialog struct {
	title string // already styled
	right string // already styled, may be empty
	rows  []string
	keys  string // plain; drawn muted
}

// render draws the dialog no wider than maxW.
func (d dialog) render(s styles, maxW int) []string {
	b := s.accent
	keys := s.muted.Render(d.keys)

	inner := lipgloss.Width(d.title) + lipgloss.Width(d.right) + 6
	inner = max(inner, lipgloss.Width(keys)+4)
	for _, r := range d.rows {
		inner = max(inner, lipgloss.Width(r)+2)
	}
	inner = min(inner, maxW-2)

	edge := func(left, label, right, end string) string {
		fill := inner - 1 - lipgloss.Width(label) - lipgloss.Width(right)
		if fill < 1 {
			label = ""
			fill = inner - 1 - lipgloss.Width(right)
		}
		if fill < 1 {
			right = ""
			fill = inner - 1
		}
		return b.Render(left+"─") + label + b.Render(strings.Repeat("─", fill)) + right + b.Render(end)
	}

	title := " " + d.title + " "
	right := ""
	if d.right != "" {
		right = " " + d.right + " " + b.Render("─")
	}
	out := []string{edge("╭", title, right, "╮")}
	for _, r := range d.rows {
		r = ansi.Truncate(r, inner-2, "…")
		pad := inner - 2 - lipgloss.Width(r)
		out = append(out, b.Render("│")+" "+r+strings.Repeat(" ", max(pad, 0))+" "+b.Render("│"))
	}
	out = append(out, edge("╰", " "+keys+" ", "", "╯"))
	return out
}

// overlay puts layer in the middle of screen (w × h), with what's behind it
// redrawn dim, so the layer is the only thing in color.
func overlay(screen string, layer []string, w, h int, dim lipgloss.Style) string {
	back := strings.Split(screen, "\n")
	layerW := 0
	for _, l := range layer {
		layerW = max(layerW, lipgloss.Width(l))
	}
	top := max((h-len(layer))/2, 0)
	left := max((w-layerW)/2, 0)

	out := make([]string, h)
	for row := 0; row < h; row++ {
		plain := ""
		if row < len(back) {
			plain = ansi.Strip(back[row])
		}
		plain = ansi.Truncate(plain, w, "")
		if pad := w - ansi.StringWidth(plain); pad > 0 {
			plain += strings.Repeat(" ", pad)
		}
		n := row - top
		if n < 0 || n >= len(layer) {
			out[row] = dim.Render(plain)
			continue
		}
		front := layer[n]
		frontW := lipgloss.Width(front)
		before := ansi.Truncate(plain, left, "")
		after := ansi.TruncateLeft(plain, left+frontW, "")
		out[row] = dim.Render(before) + front + dim.Render(after)
	}
	return strings.Join(out, "\n")
}

// openDialog asks how to open the project being chosen.
func (m Model) openDialog() []string {
	s := m.styles
	p := m.choosing
	title := s.fg.Render("open ") + s.kind[Project].Bold(true).Render(p.Name)
	right := ""
	if p.Branch != "" {
		right = s.muted.Render(" " + p.Branch)
	}

	openers := m.cfg.Projects.OpenWith
	nameW := 0
	for _, o := range openers {
		nameW = max(nameW, lipgloss.Width("["+o.KeyOrDefault()+"] "+o.Name))
	}
	var rows []string
	var keys []string
	for i, o := range openers {
		picked := i == m.choice
		mark := s.bg(s.muted, picked).Render("  ")
		if picked {
			mark = s.bg(s.accent, true).Render("❯ ")
		}
		label := s.bg(s.muted, picked).Render("["+o.KeyOrDefault()+"] ") + s.bg(s.bright.Bold(picked), picked).Render(o.Name)
		pad := nameW - lipgloss.Width("["+o.KeyOrDefault()+"] "+o.Name)
		cmd := s.bg(s.muted, picked).Render(strings.Repeat(" ", pad) + "   " + strings.Join(o.Cmd, " ") + " ")
		rows = append(rows, mark+label+cmd)
		keys = append(keys, o.KeyOrDefault())
	}
	d := dialog{
		title: title,
		right: right,
		rows:  rows,
		keys:  strings.Join(keys, "/") + " pick · ↑↓ move · enter open · esc cancel",
	}
	return margin(d.render(s, max(m.width-6, 10)))
}

// margin clears one cell around a layer, so the screen behind doesn't touch
// its border.
func margin(layer []string) []string {
	w := 0
	for _, l := range layer {
		w = max(w, lipgloss.Width(l))
	}
	blank := strings.Repeat(" ", w+2)
	out := []string{blank}
	for _, l := range layer {
		out = append(out, " "+l+" ")
	}
	return append(out, blank)
}
