package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/thiagokimo/nix-painel/internal/config"
)

type styles struct {
	fg, bright, muted, accent, border, sel lipgloss.Style
	kind                                   [4]lipgloss.Style
	group, errText, promptSign             lipgloss.Style
	selBg                                  lipgloss.Color
}

func newStyles(t config.Theme) styles {
	c := func(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
	return styles{
		fg:         c(t.Fg),
		bright:     c(t.FgBright),
		muted:      c(t.Muted),
		accent:     c(t.Accent),
		border:     c(t.Border),
		sel:        lipgloss.NewStyle().Background(lipgloss.Color(t.Selection)),
		kind:       [4]lipgloss.Style{c(t.Green), c(t.Orange), c(t.Blue), c(t.Aqua)},
		group:      c(t.Yellow),
		errText:    c(t.Red),
		promptSign: c(t.Blue),
		selBg:      lipgloss.Color(t.Selection),
	}
}

// bg puts the selection background behind a style, for the selected row.
func (s styles) bg(st lipgloss.Style, on bool) lipgloss.Style {
	if on {
		return st.Background(s.selBg)
	}
	return st
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	header := m.header()
	footer := m.footer()
	bodyH := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	body := m.body(bodyH)
	screen := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	if m.choosing != nil {
		return overlay(screen, m.openDialog(), m.width, m.height, m.styles.border)
	}
	return screen
}

func (m Model) header() string {
	s := m.styles
	left := s.accent.Bold(true).Render(m.cfg.User+"@"+m.cfg.Host) + s.promptSign.Render(" ~ ❯ ") + s.bright.Render("painel")
	right := s.muted.Render(m.now.Format("Mon 2 Jan · 15:04"))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return ansi.Truncate(left, m.width, "")
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) footer() string {
	s := m.styles
	sign := s.accent.Render("❯ ")
	if m.mode == modeNormal {
		sign = s.muted.Render("❯ ")
	}
	prompt := ansi.Truncate(sign+m.input.View(), m.width, "")

	var hint string
	switch {
	case m.status != "" && m.statusErr:
		hint = s.errText.Render(m.status)
	case m.status != "":
		hint = s.accent.Render(m.status)
	case m.mode == modePrompt:
		hint = s.muted.Render("enter run · ↑↓ pick · tab complete · esc navigate · ctrl+c quit")
	default:
		hint = s.muted.Render("hjkl move · enter run · y yazi here · / search · r reload · q quit")
	}
	return prompt + "\n" + ansi.Truncate(hint, m.width, "…")
}

// body lays out the panels in columns that fit the width: three side by side
// when there's room, two on a laptop-sized split, one stacked on a narrow pane.
func (m Model) body(h int) string {
	if h < 3 || len(m.panels) == 0 {
		return strings.Repeat("\n", max(h-1, 0))
	}
	byKind := map[Kind]*panel{}
	for _, p := range m.panels {
		byKind[p.kind] = p
	}
	pick := func(kinds ...Kind) []*panel {
		var out []*panel
		for _, k := range kinds {
			if p, ok := byKind[k]; ok {
				out = append(out, p)
			}
		}
		return out
	}

	var cols [][]*panel
	switch {
	case m.width >= 110:
		cols = [][]*panel{pick(Tool, Script), pick(Project), pick(Link)}
	case m.width >= 70:
		cols = [][]*panel{pick(Tool, Script), pick(Project, Link)}
	default:
		cols = [][]*panel{pick(Tool, Script, Project, Link)}
	}
	var nonEmpty [][]*panel
	for _, c := range cols {
		if len(c) > 0 {
			nonEmpty = append(nonEmpty, c)
		}
	}
	cols = nonEmpty

	rendered := make([]string, len(cols))
	left := m.width
	for i, col := range cols {
		w := left / (len(cols) - i)
		left -= w
		rendered[i] = m.column(col, w, h)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// column stacks panels into height h: each gets the rows it needs while they
// fit, the space left is shared out, and the last panel fills the rest.
func (m Model) column(ps []*panel, w, h int) string {
	need := make([]int, len(ps))
	for i, p := range ps {
		need[i] = len(m.lines(p, w-4)) + 2
	}
	heights := share(need, h)
	boxes := make([]string, len(ps))
	for i, p := range ps {
		boxes[i] = m.box(p, w, heights[i])
	}
	return lipgloss.JoinVertical(lipgloss.Left, boxes...)
}

// share splits total among wants, at least 3 each where possible: wants that
// fit are met, the rest split what remains evenly, and any leftover goes to
// the last one.
func share(wants []int, total int) []int {
	n := len(wants)
	out := make([]int, n)
	done := make([]bool, n)
	left, open := total, n
	for changed := true; changed && open > 0; {
		changed = false
		even := left / open
		for i, w := range wants {
			if !done[i] && max(w, 3) <= even {
				out[i], done[i] = max(w, 3), true
				left -= out[i]
				open--
				changed = true
			}
		}
	}
	for i := range out {
		if !done[i] {
			out[i] = left / max(open, 1)
		}
	}
	sum := 0
	for _, v := range out {
		sum += v
	}
	out[n-1] += total - sum
	return out
}

type line struct {
	text     string
	item     int // index into items, or -1 for a group header
	selected bool
}

// lines are a panel's rows inside its box: its items, or only the ones that
// match the prompt, with link group headers.
func (m Model) lines(p *panel, w int) []line {
	s := m.styles
	filtering := len(m.matches) > 0 || strings.TrimSpace(m.input.Value()) != ""
	selected := -1
	switch {
	case filtering && len(m.matches) > 0:
		selected = m.matches[m.sel]
	case !filtering && m.mode == modeNormal && m.panels[m.focus] == p:
		selected = p.items[p.cursor]
	}

	// While filtering, rows follow the match ranking, best first.
	order := p.items
	if filtering {
		order = nil
		for _, i := range m.matches {
			if m.items[i].Kind == p.kind {
				order = append(order, i)
			}
		}
	}

	var out []line
	group := ""
	for _, i := range order {
		it := m.items[i]
		if it.Kind == Link && it.Group != group {
			group = it.Group
			out = append(out, line{text: s.group.Render(group) + s.muted.Render("/"), item: -1})
		}
		out = append(out, line{text: m.row(it, i == selected, w), item: i, selected: i == selected})
	}
	if len(out) == 0 && filtering {
		out = append(out, line{text: s.muted.Render("no matches"), item: -1})
	}
	return out
}

func (m Model) row(it Item, selected bool, w int) string {
	s := m.styles
	mark := s.bg(s.muted, selected).Render("  ")
	if selected {
		mark = s.bg(s.accent, true).Render("❯ ")
	}
	text := mark
	if it.Alias != "" {
		text += s.bg(s.muted, selected).Render("[" + it.Alias + "] ")
	}
	text += s.bg(s.kind[it.Kind].Bold(selected), selected).Render(it.Name)
	extra := it.Desc
	if it.Kind == Project && it.Branch != "" {
		extra = " " + it.Branch
	}
	if extra != "" {
		text += s.bg(s.muted, selected).Render("  " + extra)
	}
	text = ansi.Truncate(text, w, "…")
	if pad := w - lipgloss.Width(text); pad > 0 {
		text += s.bg(s.fg, selected).Render(strings.Repeat(" ", pad))
	}
	return text
}

// box draws a panel with its title in the top border, scrolled so the
// selected row stays in view. The focused panel's border takes the accent.
func (m Model) box(p *panel, w, h int) string {
	s := m.styles
	focused := m.mode == modeNormal && m.panels[m.focus] == p
	b := s.border
	if focused {
		b = s.accent
	}
	inner := w - 2
	rows := h - 2
	ls := m.lines(p, inner-2)

	offset := 0
	for n, l := range ls {
		if l.selected && n >= rows {
			offset = n - rows + 1
		}
	}

	title := " " + s.kind[p.kind].Bold(true).Render(kindTitles[p.kind]) + s.muted.Render(fmt.Sprintf(" %d", len(p.items))) + " "
	fill := inner - 1 - lipgloss.Width(title)
	if fill < 0 {
		title, fill = "", inner-1
	}
	var sb strings.Builder
	sb.WriteString(b.Render("╭─") + title + b.Render(strings.Repeat("─", fill)+"╮"))
	for r := 0; r < rows; r++ {
		text := strings.Repeat(" ", max(inner-2, 0))
		if n := offset + r; n < len(ls) {
			text = ls[n].text
			if pad := inner - 2 - lipgloss.Width(text); pad > 0 {
				text += strings.Repeat(" ", pad)
			}
		}
		sb.WriteString("\n" + b.Render("│") + " " + text + " " + b.Render("│"))
	}
	sb.WriteString("\n" + b.Render("╰"+strings.Repeat("─", max(inner, 0))+"╯"))
	return sb.String()
}
