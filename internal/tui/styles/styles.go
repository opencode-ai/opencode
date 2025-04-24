package styles

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss/v2"
)

// Color constants
const (
	ColorBackground       = "#212121"
	ColorBackgroundDim    = "#2c2c2c"
	ColorBackgroundDarker = "#181818"
	ColorBorder           = "#4b4c5c"
	ColorForeground       = "#d3d3d3"
	ColorForegroundMid    = "#a0a0a0"
	ColorForegroundDim    = "#737373"
	ColorPrimary          = "#fab283"
	ColorWhite            = "#ffffff"
	
	// Catppuccin colors will be populated at runtime
)

var (
	light = catppuccin.Latte
	dark  = catppuccin.Mocha
	
	// Catppuccin color variables
	darkSurface0  = dark.Surface0().Hex
	darkSurface1  = dark.Surface1().Hex
	darkSurface2  = dark.Surface2().Hex
	darkOverlay0  = dark.Overlay0().Hex
	darkOverlay1  = dark.Overlay1().Hex
	darkText      = dark.Text().Hex
	darkSubtext0  = dark.Subtext0().Hex
	darkSubtext1  = dark.Subtext1().Hex
	darkBase      = dark.Base().Hex
	darkCrust     = dark.Crust().Hex
	darkBlue      = dark.Blue().Hex
	darkRed       = dark.Red().Hex
	darkGreen     = dark.Green().Hex
	darkMauve     = dark.Mauve().Hex
	darkTeal      = dark.Teal().Hex
	darkRosewater = dark.Rosewater().Hex
	darkFlamingo  = dark.Flamingo().Hex
	darkLavender  = dark.Lavender().Hex
	darkPeach     = dark.Peach().Hex
	darkYellow    = dark.Yellow().Hex
)

// NEW STYLES
var (
	Background       = lipgloss.Color(ColorBackground)
	BackgroundDim    = lipgloss.Color(ColorBackgroundDim)
	BackgroundDarker = lipgloss.Color(ColorBackgroundDarker)
	BorderColor      = lipgloss.Color(ColorBorder)

	Forground = lipgloss.Color(ColorForeground)

	ForgroundMid = lipgloss.Color(ColorForegroundMid)

	ForgroundDim = lipgloss.Color(ColorForegroundDim)

	BaseStyle = lipgloss.NewStyle().
			Background(Background).
			Foreground(Forground)

	PrimaryColor = lipgloss.Color(ColorPrimary)
)

var (
	Regular = lipgloss.NewStyle()
	Bold    = Regular.Bold(true)
	Padded  = Regular.Padding(0, 1)

	Border       = Regular.Border(lipgloss.NormalBorder())
	ThickBorder  = Regular.Border(lipgloss.ThickBorder())
	DoubleBorder = Regular.Border(lipgloss.DoubleBorder())

	// Colors
	White    = lipgloss.Color(ColorWhite)
	Surface0 = lipgloss.Color(darkSurface0)

	Overlay0 = lipgloss.Color(darkOverlay0)

	Ovelay1 = lipgloss.Color(darkOverlay1)

	Text = lipgloss.Color(darkText)

	SubText0 = lipgloss.Color(darkSubtext0)

	SubText1 = lipgloss.Color(darkSubtext1)

	LightGrey = lipgloss.Color(darkSurface0)
	Grey      = lipgloss.Color(darkSurface1)

	DarkGrey = lipgloss.Color(darkSurface2)

	Base = lipgloss.Color(darkBase)

	Crust = lipgloss.Color(darkCrust)

	Blue = lipgloss.Color(darkBlue)

	Red = lipgloss.Color(darkRed)

	Green = lipgloss.Color(darkGreen)

	Mauve = lipgloss.Color(darkMauve)

	Teal = lipgloss.Color(darkTeal)

	Rosewater = lipgloss.Color(darkRosewater)

	Flamingo = lipgloss.Color(darkFlamingo)

	Lavender = lipgloss.Color(darkLavender)

	Peach = lipgloss.Color(darkPeach)

	Yellow = lipgloss.Color(darkYellow)

	Primary   = Blue
	Secondary = Mauve

	Warning = Peach
	Error   = Red
)
