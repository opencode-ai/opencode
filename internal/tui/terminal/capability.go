package terminal

import (
	"os"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
)

// ColorProfile represents the color capabilities of the terminal
type ColorProfile int

const (
	// ProfileNoColor represents terminals with no color support
	ProfileNoColor ColorProfile = iota
	// Profile16Color represents terminals with 16 color support
	Profile16Color
	// Profile256Color represents terminals with 256 color support
	Profile256Color
	// ProfileTrueColor represents terminals with true color (16 million colors) support
	ProfileTrueColor
)

// String returns a string representation of the color profile
func (p ColorProfile) String() string {
	switch p {
	case ProfileNoColor:
		return "no-color"
	case Profile16Color:
		return "16-color"
	case Profile256Color:
		return "256-color"
	case ProfileTrueColor:
		return "true-color"
	default:
		return "unknown"
	}
}

// ChromaFormatter returns the appropriate chroma formatter for the color profile
func (p ColorProfile) ChromaFormatter() string {
	switch p {
	case ProfileNoColor:
		return "terminal"
	case Profile16Color:
		return "terminal16"
	case Profile256Color:
		return "terminal256"
	case ProfileTrueColor:
		return "terminal16m"
	default:
		return "terminal256" // Safe fallback
	}
}

// DetectColorProfile detects the color capabilities of the current terminal
func DetectColorProfile() ColorProfile {
	// Check if color is explicitly disabled
	if os.Getenv("NO_COLOR") != "" {
		return ProfileNoColor
	}

	// First try manual detection for explicit settings
	manual := detectColorProfileManual()

	// If manual detection found explicit color settings, use that
	if manual == ProfileTrueColor ||
		(manual == Profile256Color && (os.Getenv("COLORTERM") != "" || strings.Contains(strings.ToLower(os.Getenv("TERM")), "256"))) ||
		(manual == Profile16Color && os.Getenv("COLORS") != "") ||
		(manual == Profile256Color && os.Getenv("COLORS") != "") ||
		(manual == ProfileTrueColor && os.Getenv("COLORS") != "") {
		return manual
	}

	// Use termenv to detect color profile as fallback
	profile := termenv.EnvColorProfile()

	switch profile {
	case termenv.Ascii:
		// If termenv says no color but we have indicators of color support, use manual detection
		term := strings.ToLower(os.Getenv("TERM"))
		if os.Getenv("COLORTERM") != "" ||
			strings.Contains(term, "color") ||
			strings.Contains(term, "xterm") ||
			strings.Contains(term, "screen") ||
			os.Getenv("COLORS") != "" {
			return manual
		}
		// For terminals like vt100 that don't have explicit color indicators,
		// use the conservative fallback from manual detection instead of no color
		if term == "dumb" {
			return ProfileNoColor
		}
		return manual
	case termenv.ANSI:
		return Profile16Color
	case termenv.ANSI256:
		return Profile256Color
	case termenv.TrueColor:
		return ProfileTrueColor
	default:
		// Use manual detection as fallback
		return manual
	}
}

// detectColorProfileManual performs additional manual detection
func detectColorProfileManual() ColorProfile {
	// Check COLORTERM environment variable
	colorterm := strings.ToLower(os.Getenv("COLORTERM"))
	if colorterm == "truecolor" || colorterm == "24bit" {
		return ProfileTrueColor
	}

	// Check TERM environment variable
	term := strings.ToLower(os.Getenv("TERM"))

	// True color support
	if strings.Contains(term, "truecolor") ||
		strings.Contains(term, "24bit") ||
		strings.Contains(term, "direct") {
		return ProfileTrueColor
	}

	// 256 color support
	if strings.Contains(term, "256") ||
		strings.Contains(term, "256color") {
		return Profile256Color
	}

	// Check for specific terminal types known to support different color levels
	switch {
	case strings.Contains(term, "xterm"):
		// Modern xterm variants usually support 256 colors
		return Profile256Color
	case strings.Contains(term, "screen"):
		// Screen/tmux usually supports 256 colors if configured properly
		return Profile256Color
	case term == "dumb":
		return ProfileNoColor
	case strings.Contains(term, "color"):
		// If "color" is in the term name, assume at least 16 colors
		return Profile16Color
	}

	// Check COLORS environment variable
	if colorsStr := os.Getenv("COLORS"); colorsStr != "" {
		if colors, err := strconv.Atoi(colorsStr); err == nil {
			switch {
			case colors >= 16777216: // 24-bit
				return ProfileTrueColor
			case colors >= 256:
				return Profile256Color
			case colors >= 16:
				return Profile16Color
			case colors > 0:
				return Profile16Color
			default:
				return ProfileNoColor
			}
		}
	}

	// Conservative fallback - assume 256 colors for most modern terminals
	// This is safer than assuming true color which might not work
	return Profile256Color
}

// GetColorProfile returns the detected color profile for the current terminal
func GetColorProfile() ColorProfile {
	return DetectColorProfile()
}

// HasTrueColorSupport returns true if the terminal supports true color
func HasTrueColorSupport() bool {
	return GetColorProfile() == ProfileTrueColor
}

// Has256ColorSupport returns true if the terminal supports at least 256 colors
func Has256ColorSupport() bool {
	profile := GetColorProfile()
	return profile == Profile256Color || profile == ProfileTrueColor
}

// HasColorSupport returns true if the terminal supports any color
func HasColorSupport() bool {
	return GetColorProfile() != ProfileNoColor
}
