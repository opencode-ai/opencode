package styles

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/terminal"
)

var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

func getColorRGB(c lipgloss.TerminalColor) (uint8, uint8, uint8) {
	r, g, b, a := c.RGBA()

	// Un-premultiply alpha if needed
	if a > 0 && a < 0xffff {
		r = (r * 0xffff) / a
		g = (g * 0xffff) / a
		b = (b * 0xffff) / a
	}

	// Convert from 16-bit to 8-bit color
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

// rgbTo256Color converts a hex color to the closest 256-color palette index
func rgbTo256Color(hexColor string) int {
	// Remove # if present
	hexColor = strings.TrimPrefix(hexColor, "#")

	// Parse RGB values
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 0)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 0)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 0)

	// Convert to 6x6x6 color cube (colors 16-231)
	if r == g && g == b {
		// Grayscale (colors 232-255)
		gray := int(r)
		if gray < 8 {
			return 16 // Black
		} else if gray > 248 {
			return 231 // White
		} else {
			return 232 + (gray-8)/10
		}
	}

	// Convert to 6-level values
	r6 := int(r) * 5 / 255
	g6 := int(g) * 5 / 255
	b6 := int(b) * 5 / 255

	return 16 + 36*r6 + 6*g6 + b6
}

// rgbTo16ColorBg converts a hex color to the closest 16-color background ANSI code
func rgbTo16ColorBg(hexColor string) int {
	return rgbTo16ColorBase(hexColor, 40) // Background colors start at 40
}

// rgbTo16ColorBase converts a hex color to the closest 16-color ANSI code
func rgbTo16ColorBase(hexColor string, baseCode int) int {
	// Remove # if present
	hexColor = strings.TrimPrefix(hexColor, "#")

	// Parse RGB values
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 0)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 0)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 0)

	// Calculate brightness
	brightness := (r*299 + g*587 + b*114) / 1000

	// Map to closest ANSI color
	if brightness < 64 {
		return baseCode + 0 // Black
	} else if r > g && r > b {
		return baseCode + 1 // Red
	} else if g > r && g > b {
		return baseCode + 2 // Green
	} else if (r + g) > b*2 {
		return baseCode + 3 // Yellow
	} else if b > r && b > g {
		return baseCode + 4 // Blue
	} else if (r + b) > g*2 {
		return baseCode + 5 // Magenta
	} else if (g + b) > r*2 {
		return baseCode + 6 // Cyan
	} else {
		return baseCode + 7 // White
	}
}

// ForceReplaceBackgroundWithLipgloss replaces any ANSI background color codes
// in `input` with a background color that's appropriate for the terminal's capabilities.
func ForceReplaceBackgroundWithLipgloss(input string, newBgColor lipgloss.TerminalColor) string {
	// Use terminal-appropriate color format
	colorProfile := terminal.GetColorProfile()
	var newBg string

	switch colorProfile {
	case terminal.ProfileTrueColor:
		// Use 24-bit true color
		r, g, b := getColorRGB(newBgColor)
		newBg = fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
	case terminal.Profile256Color:
		// Convert to 256-color palette
		r, g, b := getColorRGB(newBgColor)
		hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
		colorIndex := rgbTo256Color(hexColor)
		newBg = fmt.Sprintf("48;5;%d", colorIndex)
	case terminal.Profile16Color:
		// Convert to 16-color palette
		r, g, b := getColorRGB(newBgColor)
		hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
		colorCode := rgbTo16ColorBg(hexColor)
		newBg = fmt.Sprintf("%d", colorCode)
	default:
		// No color support - use reverse video
		newBg = "7"
	}

	return ansiEscape.ReplaceAllStringFunc(input, func(seq string) string {
		const (
			escPrefixLen = 2 // "\x1b["
			escSuffixLen = 1 // "m"
		)

		raw := seq
		start := escPrefixLen
		end := len(raw) - escSuffixLen

		var sb strings.Builder
		// reserve enough space: original content minus bg codes + our newBg
		sb.Grow((end - start) + len(newBg) + 2)

		// scan from start..end, token by token
		for i := start; i < end; {
			// find the next ';' or end
			j := i
			for j < end && raw[j] != ';' {
				j++
			}
			token := raw[i:j]

			// fast‑path: skip "48;5;N" or "48;2;R;G;B"
			if len(token) == 2 && token[0] == '4' && token[1] == '8' {
				k := j + 1
				if k < end {
					// find next token
					l := k
					for l < end && raw[l] != ';' {
						l++
					}
					next := raw[k:l]
					if next == "5" {
						// skip "48;5;N"
						m := l + 1
						for m < end && raw[m] != ';' {
							m++
						}
						i = m + 1
						continue
					} else if next == "2" {
						// skip "48;2;R;G;B"
						m := l + 1
						for count := 0; count < 3 && m < end; count++ {
							for m < end && raw[m] != ';' {
								m++
							}
							m++
						}
						i = m
						continue
					}
				}
			}

			// decide whether to keep this token
			// manually parse ASCII digits to int
			isNum := true
			val := 0
			for p := i; p < j; p++ {
				c := raw[p]
				if c < '0' || c > '9' {
					isNum = false
					break
				}
				val = val*10 + int(c-'0')
			}
			keep := !isNum ||
				((val < 40 || val > 47) && (val < 100 || val > 107) && val != 49)

			if keep {
				if sb.Len() > 0 {
					sb.WriteByte(';')
				}
				sb.WriteString(token)
			}
			// advance past this token (and the semicolon)
			i = j + 1
		}

		// append our new background
		if sb.Len() > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(newBg)

		return "\x1b[" + sb.String() + "m"
	})
}
