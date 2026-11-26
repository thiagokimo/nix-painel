// Package ui is painel's screen: panels of tools, scripts, projects and links,
// and a prompt that finds and runs any of them.
package ui

import (
	"fmt"
	"net/url"
	"os/exec"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thiagokimo/nix-painel/internal/config"
)

type mode int

const (
	// modePrompt sends keys to the prompt; it's where painel starts.
	modePrompt mode = iota
	// modeNormal moves around the panels with vim keys.
	modeNormal
)

type panel struct {
	kind   Kind
	items  []int // indices into Model.items
	cursor int
}

type Model struct {
	cfg    config.Config
	styles styles

	items  []Item
	panels []*panel // only the kinds that have items
	focus  int

	mode    mode
	input   textinput.Model
	matches []int // indices into items, best first
	sel     int   // index into matches

	// choosing is the project painel is asking how to open, if any; choice
	// is the opener picked in projects.open_with.
	choosing *Item
	choice   int

	width, height int
	now           time.Time
	status        string
	statusErr     bool
}

type (
	tickMsg     time.Time
	execDoneMsg struct {
		name string
		err  error
	}
	openedMsg struct {
		name string
		err  error
	}
)

func New(cfg config.Config) Model {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "type a name or alias"
	m := Model{cfg: cfg, styles: newStyles(cfg.Theme), input: in, now: time.Now()}
	m.input.PlaceholderStyle = m.styles.muted
	m.input.TextStyle = m.styles.bright
	m.input.Cursor.Style = m.styles.accent
	m.input.Focus()
	m.load()
	return m
}

// load builds the items and panels again, keeping each panel's cursor where
// it can. Projects are found afresh, so new folders and branches show up.
func (m *Model) load() {
	cursors := map[Kind]int{}
	for _, p := range m.panels {
		cursors[p.kind] = p.cursor
	}
	focused := Kind(-1)
	if m.focus < len(m.panels) {
		focused = m.panels[m.focus].kind
	}

	m.items = buildItems(m.cfg)
	m.panels = nil
	for k := Tool; k <= Link; k++ {
		p := &panel{kind: k}
		for i, it := range m.items {
			if it.Kind == k {
				p.items = append(p.items, i)
			}
		}
		if len(p.items) == 0 {
			continue
		}
		p.cursor = min(cursors[k], len(p.items)-1)
		if k == focused {
			m.focus = len(m.panels)
		}
		m.panels = append(m.panels, p)
	}
	if m.focus >= len(m.panels) {
		m.focus = 0
	}
	m.refilter()
}

func (m *Model) refilter() {
	m.matches = match(m.items, m.input.Value())
	m.sel = 0
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

func tick() tea.Cmd {
	return tea.Every(time.Second*15, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		return m, tick()

	case execDoneMsg:
		m.load()
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("%s: %v", msg.name, msg.err), true)
		} else {
			m.setStatus(msg.name+" exited", false)
		}
		return m, nil

	case openedMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("couldn't open %s: %v", msg.name, msg.err), true)
		} else {
			m.setStatus("opened "+msg.name, false)
		}
		return m, nil

	case tea.KeyMsg:
		m.status = ""
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.choosing != nil {
			return m.chooseKey(msg)
		}
		if m.mode == modePrompt {
			return m.promptKey(msg)
		}
		return m.normalKey(msg)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.input.Value() != "" {
			m.input.Reset()
			m.refilter()
		} else {
			m.mode = modeNormal
			m.input.Blur()
		}
		return m, nil

	case "enter":
		q := m.input.Value()
		if q == "" {
			return m, nil
		}
		m.input.Reset()
		matches := m.matches
		sel := m.sel
		m.refilter()
		if len(matches) > 0 {
			return m.activate(m.items[matches[sel]])
		}
		if u, ok := asURL(q); ok {
			return m, m.open(u, u)
		}
		if m.cfg.Search != "" {
			return m, m.open("search "+q, m.cfg.Search+url.QueryEscape(q))
		}
		m.setStatus("nothing matches "+q, true)
		return m, nil

	case "up", "ctrl+p", "ctrl+k":
		if m.sel > 0 {
			m.sel--
		}
		return m, nil

	case "down", "ctrl+n", "ctrl+j":
		if m.sel < len(m.matches)-1 {
			m.sel++
		}
		return m, nil

	case "tab":
		if len(m.matches) > 0 {
			m.input.SetValue(m.items[m.matches[m.sel]].Name)
			m.input.CursorEnd()
			m.refilter()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refilter()
	return m, cmd
}

func (m Model) normalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.panels) == 0 {
		if msg.String() == "q" {
			return m, tea.Quit
		}
		return m, nil
	}
	p := m.panels[m.focus]
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "/", "i", ":":
		m.mode = modePrompt
		return m, m.input.Focus()
	case "j", "down":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "g", "home":
		p.cursor = 0
	case "G", "end":
		p.cursor = len(p.items) - 1
	case "l", "right", "tab":
		m.focus = (m.focus + 1) % len(m.panels)
	case "h", "left", "shift+tab":
		m.focus = (m.focus + len(m.panels) - 1) % len(m.panels)
	case "enter":
		return m.activate(m.items[p.items[p.cursor]])
	case "y":
		if it := m.items[p.items[p.cursor]]; it.Kind == Project && len(m.cfg.Projects.Alt) > 0 {
			return m, m.run(it, m.cfg.Projects.Alt)
		}
	case "r":
		m.load()
		m.setStatus("projects reloaded", false)
	}
	return m, nil
}

// activate runs an item. A project with more than one way to open it asks
// which first.
func (m Model) activate(it Item) (tea.Model, tea.Cmd) {
	if it.Kind != Project {
		return m, m.run(it, nil)
	}
	openers := m.cfg.Projects.OpenWith
	switch len(openers) {
	case 0:
		return m, nil
	case 1:
		return m, m.run(it, openers[0].Cmd)
	}
	m.choosing, m.choice = &it, 0
	return m, nil
}

// chooseKey answers "open <project> with?": an opener's key picks it, arrows
// or h/l move, enter opens, esc or q cancels.
func (m Model) chooseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	openers := m.cfg.Projects.OpenWith
	k := msg.String()
	for i, o := range openers {
		if k == o.KeyOrDefault() {
			m.choice = i
			return m.choose()
		}
	}
	switch k {
	case "esc", "q":
		m.choosing = nil
	case "enter":
		return m.choose()
	case "left", "h", "shift+tab", "up", "k":
		m.choice = (m.choice + len(openers) - 1) % len(openers)
	case "right", "l", "tab", "down", "j":
		m.choice = (m.choice + 1) % len(openers)
	}
	return m, nil
}

func (m Model) choose() (tea.Model, tea.Cmd) {
	it := *m.choosing
	m.choosing = nil
	return m, m.run(it, m.cfg.Projects.OpenWith[m.choice].Cmd)
}

func (m *Model) setStatus(s string, isErr bool) {
	m.status, m.statusErr = s, isErr
}

// The shell around a command with Pause: it runs the command, then waits for
// enter so the output can be read. $0 is the entry's name.
const pauseScript = `"$@"; s=$?; printf '\n\033[2m%s exited with %d, press enter to go back\033[0m' "$0" "$s"; read -r _`

// run hands the terminal to the item's command until it exits, or opens the
// item's link. A project runs with, inside its folder.
func (m Model) run(it Item, with []string) tea.Cmd {
	var argv []string
	dir := ""
	switch it.Kind {
	case Link:
		return m.open(it.Name, it.URL)
	case Project:
		argv = with
		dir = it.Path
	default:
		argv = it.Cmd
	}
	if len(argv) == 0 {
		return nil
	}
	if it.Pause {
		argv = append([]string{"sh", "-c", pauseScript, it.Name}, argv...)
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	name := it.Name
	return tea.ExecProcess(c, func(err error) tea.Msg { return execDoneMsg{name: name, err: err} })
}

// open starts the browser on its own session, so it outlives painel and its
// output stays off the screen.
func (m Model) open(name, target string) tea.Cmd {
	argv := append(append([]string{}, m.cfg.Browser...), target)
	return func() tea.Msg {
		c := exec.Command(argv[0], argv[1:]...)
		c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := c.Start(); err != nil {
			return openedMsg{name: name, err: err}
		}
		go c.Wait() //nolint:errcheck // nothing to do with its exit
		return openedMsg{name: name}
	}
}
