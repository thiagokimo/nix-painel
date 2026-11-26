// painel: a terminal home screen for the tools, scripts, projects and links
// used every day. Type a name or alias and press enter.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thiagokimo/nix-painel/internal/config"
	"github.com/thiagokimo/nix-painel/internal/ui"
)

var version = "dev"

func main() {
	path := flag.String("config", config.DefaultPath(), "config file")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("painel", version)
		return
	}

	cfg, err := config.Load(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "painel:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(ui.New(cfg), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "painel:", err)
		os.Exit(1)
	}
}
