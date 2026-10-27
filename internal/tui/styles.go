package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Soothing Color Palette (Nord / Tokyo Night / Catppuccin Macchiato inspired)
const (
	ColorBgBase        = "#1e1e2e" // Deep soothing background
	ColorBgCard        = "#222436" // Elevated card background
	ColorBgPill        = "#282a3e" // Subtle pill/chip background
	ColorBorderSubtle  = "#3b4261" // Understated, subtle divider & border lines
	ColorBorderFocused = "#58628b" // Focused subtle border line

	ColorTextPrimary   = "#c0caf5" // Soft high-contrast text
	ColorTextSecondary = "#9aa5ce" // Gentle secondary text
	ColorTextMuted     = "#565f89" // Subtle placeholder & hints
	ColorTextDim       = "#414868" // Darker subtle lines

	ColorPrimary   = "#7aa2f7" // Soothing primary blue
	ColorSecondary = "#bb9af7" // Calming lavender / mauve
	ColorTeal      = "#7dcfff" // Soft cyan / frost
	ColorSuccess   = "#9ece6a" // Gentle sage green (done, copied)
	ColorWarning   = "#e0af68" // Warm soft amber (streaming, loading)
	ColorDanger    = "#f7768e" // Soft coral / rose (error)

	ColorHighlightBg = "#2d3f76" // Soothing deep indigo background for highlights
	ColorHighlightFg = "#ffc777" // Warm gold/amber foreground for highlights
)

type Styles struct {
	// Legacy / Compatibility
	Header lipgloss.Style
	Label  lipgloss.Style
	Status lipgloss.Style
	Help   lipgloss.Style

	// Layout & Lines
	SubtleLine   lipgloss.Style
	CardBox      lipgloss.Style
	CardBoxFocus lipgloss.Style

	// Brand & Header Bar
	Logo         lipgloss.Style
	ModeActive   lipgloss.Style
	ModeInactive lipgloss.Style
	BadgePill    lipgloss.Style
	BadgeArrow   lipgloss.Style

	// Card Sections
	BoxLabel   lipgloss.Style
	BoxMeta    lipgloss.Style
	EmptyState lipgloss.Style

	// Highlighting
	Highlight       lipgloss.Style
	HighlightBold   lipgloss.Style
	HighlightCode   lipgloss.Style
	HighlightMatch  lipgloss.Style
	SectionTitle    lipgloss.Style
	PhoneticTag     lipgloss.Style
	PosBadge        lipgloss.Style
	TranslationWord lipgloss.Style
	ExampleSource   lipgloss.Style
	ExampleTarget   lipgloss.Style
	BulletPoint     lipgloss.Style
	ArrowSymbol     lipgloss.Style

	// Status & Notifications
	StatusSuccess lipgloss.Style
	StatusWarning lipgloss.Style
	StatusDanger  lipgloss.Style

	// Keycaps / Footer
	KeycapKey  lipgloss.Style
	KeycapDesc lipgloss.Style

	// Modal / Overlay Dialog
	ModalOverlay lipgloss.Style
	ModalBox     lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalHint    lipgloss.Style
}

func DefaultStyles() Styles {
	subtleBorder := lipgloss.RoundedBorder()

	return Styles{
		// Legacy / basic styles
		Header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPrimary)),
		Label:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorTextSecondary)),
		Status: lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)),
		Help:   lipgloss.NewStyle().Foreground(lipgloss.Color(ColorTextMuted)),

		// Layout & Understated Lines
		SubtleLine: lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorderSubtle)),
		CardBox: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorBorderSubtle)).
			Padding(0, 1),
		CardBoxFocus: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorBorderFocused)).
			Padding(0, 1),

		// Brand & Header
		Logo: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)),
		ModeActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBgBase)).
			Background(lipgloss.Color(ColorPrimary)).
			Padding(0, 1),
		ModeInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1),
		BadgePill: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextSecondary)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1),
		BadgeArrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Bold(true),

		// Card Sections
		BoxLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorTeal)),
		BoxMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		EmptyState: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color(ColorTextMuted)),

		// Text & Section Highlighting
		Highlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorHighlightFg)).
			Background(lipgloss.Color(ColorHighlightBg)).
			Padding(0, 1),
		HighlightBold: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorTextPrimary)),
		HighlightCode: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1),
		HighlightMatch: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorHighlightFg)).
			Background(lipgloss.Color(ColorHighlightBg)),
		SectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorSecondary)),
		PhoneticTag: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1),
		PosBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorSecondary)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1),
		TranslationWord: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorSuccess)),
		ExampleSource: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextPrimary)).
			Italic(true),
		ExampleTarget: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)),
		BulletPoint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true),
		ArrowSymbol: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true),

		// Status
		StatusSuccess: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true),
		StatusWarning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)),
		StatusDanger: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDanger)).
			Bold(true),

		// Footer Keycaps
		KeycapKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1).
			Bold(true),
		KeycapDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),

		// Modal
		ModalOverlay: lipgloss.NewStyle().
			Padding(1, 2),
		ModalBox: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorPrimary)).
			Padding(1, 2),
		ModalTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)),
		ModalHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Italic(true),
	}
}
