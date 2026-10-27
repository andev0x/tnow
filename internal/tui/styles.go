package tui

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Header lipgloss.Style
	Label  lipgloss.Style
	Status lipgloss.Style
	Help   lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Label:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")),
		Status: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		Help:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}
