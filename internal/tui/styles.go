package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette inspired by Tokyo Night / Catppuccin Macchiato. Every surface has a
// dedicated opaque background so text never bleeds through from the terminal
// or from a panel below an overlay.
var (
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
