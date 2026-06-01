package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Primary color palette
	Primary    = lipgloss.Color("#E9A823") // warm gold (Claude Code style)
	Secondary  = lipgloss.Color("#D97706") // deep amber
	Success    = lipgloss.Color("#10B981") // green
	Error      = lipgloss.Color("#EF4444") // red
	Warning    = lipgloss.Color("#F59E0B") // amber
	Muted      = lipgloss.Color("#6B7280") // gray
	BrightText = lipgloss.Color("#F9FAFB")
	DimText    = lipgloss.Color("#9CA3AF")

	// Title style — large, bold, colored
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	// Subtitle
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			MarginBottom(1)

	// Status indicators
	SuccessStyle = lipgloss.NewStyle().Foreground(Success).SetString("✓")
	ErrorStyle   = lipgloss.NewStyle().Foreground(Error).SetString("✗")
	PendingStyle = lipgloss.NewStyle().Foreground(Muted).SetString("…")

	// Menu item styles
	ActiveItemStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			PaddingLeft(2)

	InactiveItemStyle = lipgloss.NewStyle().
				Foreground(DimText).
				PaddingLeft(2)

	// Container
	ContainerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1)
)

// RenderTitle renders the app title with optional subtitle.
func RenderTitle() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(Primary).Render("MyMeetily"),
		lipgloss.NewStyle().Foreground(DimText).Render("本地离线会议录音转文字 + 结构化会议纪要"),
	)
}

// RenderHelp renders keyboard help text.
func RenderHelp(keys ...string) string {
	help := ""
	for i := 0; i < len(keys); i += 2 {
		if i > 0 {
			help += "  "
		}
		help += lipgloss.NewStyle().Foreground(Primary).Render(keys[i]) +
			" " + lipgloss.NewStyle().Foreground(DimText).Render(keys[i+1])
	}
	return HelpStyle.Render(help)
}
