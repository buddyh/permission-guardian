package tui

import "charm.land/lipgloss/v2"

// ═══════════════════════════════════════════════════════════════════════════
// Color Palette
// ═══════════════════════════════════════════════════════════════════════════

var (
	// Base colors
	ColorBg       = lipgloss.Color("#1e2139")
	ColorSurface  = lipgloss.Color("#282c47")
	ColorSelected = lipgloss.Color("#3d4466")
	ColorBorder   = lipgloss.Color("#5a6178")
	ColorTextPri  = lipgloss.Color("#e8eaee")
	ColorTextSec  = lipgloss.Color("#b8bcc8")
	ColorTextDim  = lipgloss.Color("#8890a0")

	// Semantic colors
	ColorSuccess = lipgloss.Color("#5eead4") // Approved/healthy
	ColorWarning = lipgloss.Color("#fbbf24") // Waiting/pending
	ColorError   = lipgloss.Color("#ff6b6b") // Denied/error
	ColorActive  = lipgloss.Color("#a78bfa") // Working/active
	ColorInfo    = lipgloss.Color("#60a5fa") // Info/neutral
	ColorAccent  = lipgloss.Color("#f472b6") // Highlight/accent

	// Zebra stripe
	ColorZebra = lipgloss.Color("#252840")
)

// ═══════════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════════

var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Background(ColorBg)

	// Logo styles
	LogoStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	LogoSubStyle = lipgloss.NewStyle().
			Foreground(ColorTextSec)

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorTextPri).
			Padding(0, 2)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Foreground(ColorTextSec).
			Bold(true).
			Padding(0, 1)

	// Session list styles
	SessionNameStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true)

	SessionSelectedStyle = lipgloss.NewStyle().
				Background(ColorSelected).
				Foreground(ColorTextPri).
				Bold(true)

	// Status styles
	StatusWaiting = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	StatusWorking = lipgloss.NewStyle().
			Foreground(ColorActive)

	StatusIdle = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	StatusApproved = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StatusDenied = lipgloss.NewStyle().
			Foreground(ColorError)

	StatusAccent = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StatusAuto = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Detail panel styles
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(ColorTextSec)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ColorTextPri)

	// Preview styles
	PreviewStyle = lipgloss.NewStyle().
			Foreground(ColorTextPri).
			Background(ColorSurface).
			Padding(0, 1)

	// Help bar styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Padding(0, 2)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	// Divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)

	// Separator style
	SepStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)
)
