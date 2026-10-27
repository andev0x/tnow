package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andev0x/tnow/internal/app"
	"github.com/andev0x/tnow/internal/cli"
	"github.com/andev0x/tnow/internal/config"
	"github.com/andev0x/tnow/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	a, err := app.New(cfg, "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := os.Args[1:]

	// determine stdin
	var stdin *os.File
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		stdin = os.Stdin
	}

	code := cli.Run(a, args, stdin, os.Stdout, os.Stderr)
	if code == -1 {
		// launch TUI
		m := tui.NewModel(a)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(code)
}
