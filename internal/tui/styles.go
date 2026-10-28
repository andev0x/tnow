package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette inspired by Tokyo Night / Catppuccin Macchiato. Every surface has a
// dedicated opaque background so text never bleeds through from the terminal
// or from a panel below an overlay.
const (
	ColorBgBase      = "#16161e" // Full-screen canvas
	ColorBgPanel     = "#1e2030" // Input / output panels
	ColorBgRaised    = "#24273a" // Floating command palette and dialogs
	ColorBgBar       = "#1b1d29" // Bottom status / shortcut bar
	ColorBgPill      = "#2a2e42" // Inline badges
	ColorBgSelection = "#2f3b61" // Selected command row
	ColorBgShadow    = "#0f1017" // Depth under floating surfaces
	ColorBgCard      = ColorBgPanel

	ColorBorderSubtle  = "#2f3549"
	ColorBorderFocused = "#7dcfff" // Teal focus accent
	ColorBorderIdle    = "#3b4261"

	ColorTextPrimary   = "#c0caf5"
	ColorTextSecondary = "#a9b1d6"
	ColorTextMuted     = "#565f89"
	ColorTextDim       = "#414868"

	ColorPrimary   = "#7aa2f7"
	ColorSecondary = "#bb9af7"
	ColorTeal      = "#7dcfff"
	ColorSuccess   = "#9ece6a"
	ColorWarning   = "#e0af68"
	ColorDanger    = "#f7768e"

	ColorHighlightBg = "#2d3f76"
	ColorHighlightFg = "#ffc777"
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

	// Brand & Header
	Logo         lipgloss.Style
	ModeActive   lipgloss.Style
	ModeInactive lipgloss.Style
	HeaderMeta   lipgloss.Style
	BadgePill    lipgloss.Style
	BadgeArrow   lipgloss.Style

	// Card Sections
	BoxLabel      lipgloss.Style
	PanelTitle    lipgloss.Style
	PanelTitleOn  lipgloss.Style
	PanelMeta     lipgloss.Style
	BoxMeta       lipgloss.Style
	EmptyState    lipgloss.Style
	SectionHeader lipgloss.Style

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
	StatusNeutral lipgloss.Style

	// Keycaps / Footer
	KeycapKey  lipgloss.Style
	KeycapDesc lipgloss.Style
	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style

	// Modal / Overlay Dialog
	ModalOverlay lipgloss.Style
	ModalBox     lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalHint    lipgloss.Style

	// Autocomplete / Suggestions
	SuggestBox     lipgloss.Style
	SuggestActive  lipgloss.Style
	SuggestIdle    lipgloss.Style
	Ghost          lipgloss.Style
	BadgeExcellent lipgloss.Style

	// Command palette
	PopupTitle    lipgloss.Style
	PopupMeta     lipgloss.Style
	PopupKey      lipgloss.Style
	PopupDesc     lipgloss.Style
	PopupPointer  lipgloss.Style
	PopupSelected lipgloss.Style
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
			Background(lipgloss.Color(ColorBgPanel)).
			Padding(0, 1),
		CardBoxFocus: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorBorderFocused)).
			Background(lipgloss.Color(ColorBgPanel)).
			Padding(0, 1),

		// Brand & Header
		Logo: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorTeal)),
		ModeActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBgBase)).
			Background(lipgloss.Color(ColorPrimary)).
			Padding(0, 1),
		ModeInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Padding(0, 1),
		HeaderMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextSecondary)),
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
		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorTextSecondary)),
		PanelTitleOn: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBorderFocused)),
		PanelMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		BoxMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		EmptyState: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color(ColorTextMuted)),
		SectionHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorSecondary)),

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
		StatusNeutral: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),

		// Footer Keycaps
		KeycapKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Background(lipgloss.Color(ColorBgPill)).
			Padding(0, 1).
			Bold(true),
		KeycapDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		FooterKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTeal)).
			Bold(true),
		FooterDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),

		// Modal
		ModalOverlay: lipgloss.NewStyle().
			Padding(1, 2),
		ModalBox: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorBorderFocused)).
			Background(lipgloss.Color(ColorBgRaised)).
			Padding(1, 2),
		ModalTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)),
		ModalHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Italic(true),

		// Autocomplete / Suggestions (minimal, stable height)
		SuggestBox: lipgloss.NewStyle().
			Border(subtleBorder).
			BorderForeground(lipgloss.Color(ColorBorderSubtle)).
			Background(lipgloss.Color(ColorBgRaised)).
			Padding(0, 1),
		SuggestActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextPrimary)).
			Background(lipgloss.Color(ColorBgSelection)).
			Bold(true),
		SuggestIdle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextPrimary)),
		Ghost: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)).
			Italic(true),
		BadgeExcellent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true),

		// Command palette
		PopupTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBorderFocused)),
		PopupMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		PopupKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPrimary)),
		PopupDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorTextMuted)),
		PopupPointer: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBorderFocused)),
		PopupSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorTextPrimary)).
			Background(lipgloss.Color(ColorBgSelection)),
	}
}
