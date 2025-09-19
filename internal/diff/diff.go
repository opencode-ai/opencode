package diff

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/tui/terminal"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// -------------------------------------------------------------------------
// Core Types
// -------------------------------------------------------------------------

// LineType represents the kind of line in a diff.
type LineType int

const (
	LineContext LineType = iota // Line exists in both files
	LineAdded                   // Line added in the new file
	LineRemoved                 // Line removed from the old file
)

// Segment represents a portion of a line for intra-line highlighting
type Segment struct {
	Start int
	End   int
	Type  LineType
	Text  string
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	OldLineNo int       // Line number in old file (0 for added lines)
	NewLineNo int       // Line number in new file (0 for removed lines)
	Kind      LineType  // Type of line (added, removed, context)
	Content   string    // Content of the line
	Segments  []Segment // Segments for intraline highlighting
}

// Hunk represents a section of changes in a diff
type Hunk struct {
	Header string
	Lines  []DiffLine
}

// DiffResult contains the parsed result of a diff
type DiffResult struct {
	OldFile string
	NewFile string
	Hunks   []Hunk
}

// linePair represents a pair of lines for side-by-side display
type linePair struct {
	left  *DiffLine
	right *DiffLine
}

// -------------------------------------------------------------------------
// Parse Configuration
// -------------------------------------------------------------------------

// ParseConfig configures the behavior of diff parsing
type ParseConfig struct {
	ContextSize int // Number of context lines to include
}

// ParseOption modifies a ParseConfig
type ParseOption func(*ParseConfig)

// WithContextSize sets the number of context lines to include
func WithContextSize(size int) ParseOption {
	return func(p *ParseConfig) {
		if size >= 0 {
			p.ContextSize = size
		}
	}
}

// -------------------------------------------------------------------------
// Side-by-Side Configuration
// -------------------------------------------------------------------------

// SideBySideConfig configures the rendering of side-by-side diffs
type SideBySideConfig struct {
	TotalWidth int
}

// SideBySideOption modifies a SideBySideConfig
type SideBySideOption func(*SideBySideConfig)

// NewSideBySideConfig creates a SideBySideConfig with default values
func NewSideBySideConfig(opts ...SideBySideOption) SideBySideConfig {
	config := SideBySideConfig{
		TotalWidth: 160, // Default width for side-by-side view
	}

	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// WithTotalWidth sets the total width for side-by-side view
func WithTotalWidth(width int) SideBySideOption {
	return func(s *SideBySideConfig) {
		if width > 0 {
			s.TotalWidth = width
		}
	}
}

// -------------------------------------------------------------------------
// Diff Parsing
// -------------------------------------------------------------------------

// ParseUnifiedDiff parses a unified diff format string into structured data
func ParseUnifiedDiff(diff string) (DiffResult, error) {
	var result DiffResult
	var currentHunk *Hunk

	hunkHeaderRe := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)
	lines := strings.Split(diff, "\n")

	var oldLine, newLine int
	inFileHeader := true

	for _, line := range lines {
		// Parse file headers
		if inFileHeader {
			if strings.HasPrefix(line, "--- a/") {
				result.OldFile = strings.TrimPrefix(line, "--- a/")
				continue
			}
			if strings.HasPrefix(line, "+++ b/") {
				result.NewFile = strings.TrimPrefix(line, "+++ b/")
				inFileHeader = false
				continue
			}
		}

		// Parse hunk headers
		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				result.Hunks = append(result.Hunks, *currentHunk)
			}
			currentHunk = &Hunk{
				Header: line,
				Lines:  []DiffLine{},
			}

			oldStart, _ := strconv.Atoi(matches[1])
			newStart, _ := strconv.Atoi(matches[3])
			oldLine = oldStart
			newLine = newStart
			continue
		}

		// Ignore "No newline at end of file" markers
		if strings.HasPrefix(line, "\\ No newline at end of file") {
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Process the line based on its prefix
		if len(line) > 0 {
			switch line[0] {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: 0,
					NewLineNo: newLine,
					Kind:      LineAdded,
					Content:   line[1:],
				})
				newLine++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: oldLine,
					NewLineNo: 0,
					Kind:      LineRemoved,
					Content:   line[1:],
				})
				oldLine++
			default:
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					OldLineNo: oldLine,
					NewLineNo: newLine,
					Kind:      LineContext,
					Content:   line,
				})
				oldLine++
				newLine++
			}
		} else {
			// Handle empty lines
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				OldLineNo: oldLine,
				NewLineNo: newLine,
				Kind:      LineContext,
				Content:   "",
			})
			oldLine++
			newLine++
		}
	}

	// Add the last hunk if there is one
	if currentHunk != nil {
		result.Hunks = append(result.Hunks, *currentHunk)
	}

	return result, nil
}

// HighlightIntralineChanges updates lines in a hunk to show character-level differences
func HighlightIntralineChanges(h *Hunk) {
	var updated []DiffLine
	dmp := diffmatchpatch.New()

	for i := 0; i < len(h.Lines); i++ {
		// Look for removed line followed by added line
		if i+1 < len(h.Lines) &&
			h.Lines[i].Kind == LineRemoved &&
			h.Lines[i+1].Kind == LineAdded {

			oldLine := h.Lines[i]
			newLine := h.Lines[i+1]

			// Find character-level differences
			patches := dmp.DiffMain(oldLine.Content, newLine.Content, false)
			patches = dmp.DiffCleanupSemantic(patches)
			patches = dmp.DiffCleanupMerge(patches)
			patches = dmp.DiffCleanupEfficiency(patches)

			segments := make([]Segment, 0)

			removeStart := 0
			addStart := 0
			for _, patch := range patches {
				switch patch.Type {
				case diffmatchpatch.DiffDelete:
					segments = append(segments, Segment{
						Start: removeStart,
						End:   removeStart + len(patch.Text),
						Type:  LineRemoved,
						Text:  patch.Text,
					})
					removeStart += len(patch.Text)
				case diffmatchpatch.DiffInsert:
					segments = append(segments, Segment{
						Start: addStart,
						End:   addStart + len(patch.Text),
						Type:  LineAdded,
						Text:  patch.Text,
					})
					addStart += len(patch.Text)
				default:
					// Context text, no highlighting needed
					removeStart += len(patch.Text)
					addStart += len(patch.Text)
				}
			}
			oldLine.Segments = segments
			newLine.Segments = segments

			updated = append(updated, oldLine, newLine)
			i++ // Skip the next line as we've already processed it
		} else {
			updated = append(updated, h.Lines[i])
		}
	}

	h.Lines = updated
}

// pairLines converts a flat list of diff lines to pairs for side-by-side display
func pairLines(lines []DiffLine) []linePair {
	var pairs []linePair
	i := 0

	for i < len(lines) {
		switch lines[i].Kind {
		case LineRemoved:
			// Check if the next line is an addition, if so pair them
			if i+1 < len(lines) && lines[i+1].Kind == LineAdded {
				pairs = append(pairs, linePair{left: &lines[i], right: &lines[i+1]})
				i += 2
			} else {
				pairs = append(pairs, linePair{left: &lines[i], right: nil})
				i++
			}
		case LineAdded:
			pairs = append(pairs, linePair{left: nil, right: &lines[i]})
			i++
		case LineContext:
			pairs = append(pairs, linePair{left: &lines[i], right: &lines[i]})
			i++
		}
	}

	return pairs
}

// -------------------------------------------------------------------------
// Syntax Highlighting
// -------------------------------------------------------------------------

// SyntaxHighlight applies syntax highlighting to text based on file extension
func SyntaxHighlight(w io.Writer, source, fileName, formatter string, bg lipgloss.TerminalColor) error {
	t := theme.CurrentTheme()

	// Determine the language lexer to use
	l := lexers.Match(fileName)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// Get the formatter
	f := formatters.Get(formatter)
	if f == nil {
		f = formatters.Fallback
	}

	// Dynamic theme based on current theme values
	syntaxThemeXml := fmt.Sprintf(`
	<style name="opencode-theme">
	<!-- Base colors -->
	<entry type="Background" style="bg:%s"/>
	<entry type="Text" style="%s"/>
	<entry type="Other" style="%s"/>
	<entry type="Error" style="%s"/>
	<!-- Keywords -->
	<entry type="Keyword" style="%s"/>
	<entry type="KeywordConstant" style="%s"/>
	<entry type="KeywordDeclaration" style="%s"/>
	<entry type="KeywordNamespace" style="%s"/>
	<entry type="KeywordPseudo" style="%s"/>
	<entry type="KeywordReserved" style="%s"/>
	<entry type="KeywordType" style="%s"/>
	<!-- Names -->
	<entry type="Name" style="%s"/>
	<entry type="NameAttribute" style="%s"/>
	<entry type="NameBuiltin" style="%s"/>
	<entry type="NameBuiltinPseudo" style="%s"/>
	<entry type="NameClass" style="%s"/>
	<entry type="NameConstant" style="%s"/>
	<entry type="NameDecorator" style="%s"/>
	<entry type="NameEntity" style="%s"/>
	<entry type="NameException" style="%s"/>
	<entry type="NameFunction" style="%s"/>
	<entry type="NameLabel" style="%s"/>
	<entry type="NameNamespace" style="%s"/>
	<entry type="NameOther" style="%s"/>
	<entry type="NameTag" style="%s"/>
	<entry type="NameVariable" style="%s"/>
	<entry type="NameVariableClass" style="%s"/>
	<entry type="NameVariableGlobal" style="%s"/>
	<entry type="NameVariableInstance" style="%s"/>
	<!-- Literals -->
	<entry type="Literal" style="%s"/>
	<entry type="LiteralDate" style="%s"/>
	<entry type="LiteralString" style="%s"/>
	<entry type="LiteralStringBacktick" style="%s"/>
	<entry type="LiteralStringChar" style="%s"/>
	<entry type="LiteralStringDoc" style="%s"/>
	<entry type="LiteralStringDouble" style="%s"/>
	<entry type="LiteralStringEscape" style="%s"/>
	<entry type="LiteralStringHeredoc" style="%s"/>
	<entry type="LiteralStringInterpol" style="%s"/>
	<entry type="LiteralStringOther" style="%s"/>
	<entry type="LiteralStringRegex" style="%s"/>
	<entry type="LiteralStringSingle" style="%s"/>
	<entry type="LiteralStringSymbol" style="%s"/>
	<!-- Numbers -->
	<entry type="LiteralNumber" style="%s"/>
	<entry type="LiteralNumberBin" style="%s"/>
	<entry type="LiteralNumberFloat" style="%s"/>
	<entry type="LiteralNumberHex" style="%s"/>
	<entry type="LiteralNumberInteger" style="%s"/>
	<entry type="LiteralNumberIntegerLong" style="%s"/>
	<entry type="LiteralNumberOct" style="%s"/>
	<!-- Operators -->
	<entry type="Operator" style="%s"/>
	<entry type="OperatorWord" style="%s"/>
	<entry type="Punctuation" style="%s"/>
	<!-- Comments -->
	<entry type="Comment" style="%s"/>
	<entry type="CommentHashbang" style="%s"/>
	<entry type="CommentMultiline" style="%s"/>
	<entry type="CommentSingle" style="%s"/>
	<entry type="CommentSpecial" style="%s"/>
	<entry type="CommentPreproc" style="%s"/>
	<!-- Generic styles -->
	<entry type="Generic" style="%s"/>
	<entry type="GenericDeleted" style="%s"/>
	<entry type="GenericEmph" style="italic %s"/>
	<entry type="GenericError" style="%s"/>
	<entry type="GenericHeading" style="bold %s"/>
	<entry type="GenericInserted" style="%s"/>
	<entry type="GenericOutput" style="%s"/>
	<entry type="GenericPrompt" style="%s"/>
	<entry type="GenericStrong" style="bold %s"/>
	<entry type="GenericSubheading" style="bold %s"/>
	<entry type="GenericTraceback" style="%s"/>
	<entry type="GenericUnderline" style="underline"/>
	<entry type="TextWhitespace" style="%s"/>
</style>
`,
		getColor(t.Background()), // Background
		getColor(t.Text()),       // Text
		getColor(t.Text()),       // Other
		getColor(t.Error()),      // Error

		getColor(t.SyntaxKeyword()), // Keyword
		getColor(t.SyntaxKeyword()), // KeywordConstant
		getColor(t.SyntaxKeyword()), // KeywordDeclaration
		getColor(t.SyntaxKeyword()), // KeywordNamespace
		getColor(t.SyntaxKeyword()), // KeywordPseudo
		getColor(t.SyntaxKeyword()), // KeywordReserved
		getColor(t.SyntaxType()),    // KeywordType

		getColor(t.Text()),           // Name
		getColor(t.SyntaxVariable()), // NameAttribute
		getColor(t.SyntaxType()),     // NameBuiltin
		getColor(t.SyntaxVariable()), // NameBuiltinPseudo
		getColor(t.SyntaxType()),     // NameClass
		getColor(t.SyntaxVariable()), // NameConstant
		getColor(t.SyntaxFunction()), // NameDecorator
		getColor(t.SyntaxVariable()), // NameEntity
		getColor(t.SyntaxType()),     // NameException
		getColor(t.SyntaxFunction()), // NameFunction
		getColor(t.Text()),           // NameLabel
		getColor(t.SyntaxType()),     // NameNamespace
		getColor(t.SyntaxVariable()), // NameOther
		getColor(t.SyntaxKeyword()),  // NameTag
		getColor(t.SyntaxVariable()), // NameVariable
		getColor(t.SyntaxVariable()), // NameVariableClass
		getColor(t.SyntaxVariable()), // NameVariableGlobal
		getColor(t.SyntaxVariable()), // NameVariableInstance

		getColor(t.SyntaxString()), // Literal
		getColor(t.SyntaxString()), // LiteralDate
		getColor(t.SyntaxString()), // LiteralString
		getColor(t.SyntaxString()), // LiteralStringBacktick
		getColor(t.SyntaxString()), // LiteralStringChar
		getColor(t.SyntaxString()), // LiteralStringDoc
		getColor(t.SyntaxString()), // LiteralStringDouble
		getColor(t.SyntaxString()), // LiteralStringEscape
		getColor(t.SyntaxString()), // LiteralStringHeredoc
		getColor(t.SyntaxString()), // LiteralStringInterpol
		getColor(t.SyntaxString()), // LiteralStringOther
		getColor(t.SyntaxString()), // LiteralStringRegex
		getColor(t.SyntaxString()), // LiteralStringSingle
		getColor(t.SyntaxString()), // LiteralStringSymbol

		getColor(t.SyntaxNumber()), // LiteralNumber
		getColor(t.SyntaxNumber()), // LiteralNumberBin
		getColor(t.SyntaxNumber()), // LiteralNumberFloat
		getColor(t.SyntaxNumber()), // LiteralNumberHex
		getColor(t.SyntaxNumber()), // LiteralNumberInteger
		getColor(t.SyntaxNumber()), // LiteralNumberIntegerLong
		getColor(t.SyntaxNumber()), // LiteralNumberOct

		getColor(t.SyntaxOperator()),    // Operator
		getColor(t.SyntaxKeyword()),     // OperatorWord
		getColor(t.SyntaxPunctuation()), // Punctuation

		getColor(t.SyntaxComment()), // Comment
		getColor(t.SyntaxComment()), // CommentHashbang
		getColor(t.SyntaxComment()), // CommentMultiline
		getColor(t.SyntaxComment()), // CommentSingle
		getColor(t.SyntaxComment()), // CommentSpecial
		getColor(t.SyntaxKeyword()), // CommentPreproc

		getColor(t.Text()),      // Generic
		getColor(t.Error()),     // GenericDeleted
		getColor(t.Text()),      // GenericEmph
		getColor(t.Error()),     // GenericError
		getColor(t.Text()),      // GenericHeading
		getColor(t.Success()),   // GenericInserted
		getColor(t.TextMuted()), // GenericOutput
		getColor(t.Text()),      // GenericPrompt
		getColor(t.Text()),      // GenericStrong
		getColor(t.Text()),      // GenericSubheading
		getColor(t.Error()),     // GenericTraceback
		getColor(t.Text()),      // TextWhitespace
	)

	r := strings.NewReader(syntaxThemeXml)
	style := chroma.MustNewXMLStyle(r)

	// Modify the style to use the provided background
	s, err := style.Builder().Transform(
		func(t chroma.StyleEntry) chroma.StyleEntry {
			r, g, b, _ := bg.RGBA()
			t.Background = chroma.NewColour(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			return t
		},
	).Build()
	if err != nil {
		s = styles.Fallback
	}

	// Tokenize and format
	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}

	return f.Format(w, s, it)
}

// getColor returns the appropriate hex color string based on terminal background
func getColor(adaptiveColor lipgloss.AdaptiveColor) string {
	if lipgloss.HasDarkBackground() {
		return adaptiveColor.Dark
	}
	return adaptiveColor.Light
}

// highlightLine applies syntax highlighting to a single line
func highlightLine(fileName string, line string, bg lipgloss.TerminalColor) string {
	var buf bytes.Buffer
	// Use terminal-appropriate formatter instead of hardcoding terminal16m
	colorProfile := terminal.GetColorProfile()
	formatter := colorProfile.ChromaFormatter()
	err := SyntaxHighlight(&buf, line, fileName, formatter, bg)
	if err != nil {
		return line
	}
	return buf.String()
}

// createStyles generates the lipgloss styles needed for rendering diffs
func createStyles(t theme.Theme) (removedLineStyle, addedLineStyle, contextLineStyle, lineNumberStyle lipgloss.Style) {
	removedLineStyle = lipgloss.NewStyle().Background(t.DiffRemovedBg())
	addedLineStyle = lipgloss.NewStyle().Background(t.DiffAddedBg())
	contextLineStyle = lipgloss.NewStyle().Background(t.DiffContextBg())
	lineNumberStyle = lipgloss.NewStyle().Foreground(t.DiffLineNumber())

	return
}

// -------------------------------------------------------------------------
// Rendering Functions
// -------------------------------------------------------------------------

func lipglossToHex(color lipgloss.Color) string {
	r, g, b, a := color.RGBA()

	// Scale uint32 values (0-65535) to uint8 (0-255).
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(a >> 8)

	return fmt.Sprintf("#%02x%02x%02x%02x", r8, g8, b8, a8)
}

// applyHighlighting applies intra-line highlighting to a piece of text
func applyHighlighting(content string, segments []Segment, segmentType LineType, highlightBg lipgloss.AdaptiveColor) string {
	// Find all ANSI sequences in the content
	ansiRegex := regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-9?]*(?:;[0-9?]*)*[@-~])`)
	ansiMatches := ansiRegex.FindAllStringIndex(content, -1)

	// Build a mapping of visible character positions to their actual indices
	visibleIdx := 0
	ansiSequences := make(map[int]string)
	lastAnsiSeq := "\x1b[0m" // Default reset sequence

	for i := 0; i < len(content); {
		isAnsi := false
		for _, match := range ansiMatches {
			if match[0] == i {
				ansiSequences[visibleIdx] = content[match[0]:match[1]]
				lastAnsiSeq = content[match[0]:match[1]]
				i = match[1]
				isAnsi = true
				break
			}
		}
		if isAnsi {
			continue
		}

		// For non-ANSI positions, store the last ANSI sequence
		if _, exists := ansiSequences[visibleIdx]; !exists {
			ansiSequences[visibleIdx] = lastAnsiSeq
		}
		visibleIdx++
		i++
	}

	// Apply highlighting
	var sb strings.Builder
	inSelection := false
	currentPos := 0

	// Get the appropriate color based on terminal background and capabilities
	bgColorHex := getColor(highlightBg)
	fgColorHex := getColor(theme.CurrentTheme().Background())

	// Generate appropriate color sequences based on terminal capabilities
	colorProfile := terminal.GetColorProfile()
	var bgColorSeq, fgColorSeq string

	switch colorProfile {
	case terminal.ProfileTrueColor:
		// Use 24-bit true color
		bgColor := lipgloss.Color(bgColorHex)
		fgColor := lipgloss.Color(fgColorHex)
		r, g, b, _ := fgColor.RGBA()
		fgColorSeq = fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
		r, g, b, _ = bgColor.RGBA()
		bgColorSeq = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r>>8, g>>8, b>>8)
	case terminal.Profile256Color:
		// Use 256-color palette - convert RGB to closest 256-color index
		bgColorSeq = fmt.Sprintf("\x1b[48;5;%dm", rgbTo256Color(bgColorHex))
		fgColorSeq = fmt.Sprintf("\x1b[38;5;%dm", rgbTo256Color(fgColorHex))
	case terminal.Profile16Color:
		// Use basic 16-color palette - map to closest ANSI color
		bgColorSeq = fmt.Sprintf("\x1b[%dm", rgbTo16ColorBg(bgColorHex))
		fgColorSeq = fmt.Sprintf("\x1b[%dm", rgbTo16ColorFg(fgColorHex))
	default:
		// No color support - use default highlighting
		bgColorSeq = "\x1b[7m" // Reverse video
		fgColorSeq = ""
	}

	for i := 0; i < len(content); {
		// Check if we're at an ANSI sequence
		isAnsi := false
		for _, match := range ansiMatches {
			if match[0] == i {
				sb.WriteString(content[match[0]:match[1]]) // Preserve ANSI sequence
				i = match[1]
				isAnsi = true
				break
			}
		}
		if isAnsi {
			continue
		}

		// Check for segment boundaries
		for _, seg := range segments {
			if seg.Type == segmentType {
				if currentPos == seg.Start {
					inSelection = true
				}
				if currentPos == seg.End {
					inSelection = false
				}
			}
		}

		// Get current character
		char := string(content[i])

		if inSelection {
			// Get the current styling
			currentStyle := ansiSequences[currentPos]

			// Apply foreground and background highlight
			sb.WriteString(fgColorSeq)
			sb.WriteString(bgColorSeq)
			sb.WriteString(char)

			// Reset colors appropriately
			if colorProfile >= terminal.Profile256Color {
				sb.WriteString("\x1b[39m") // Reset foreground
				sb.WriteString("\x1b[49m") // Reset background
			} else {
				sb.WriteString("\x1b[0m") // Full reset for simpler terminals
			}

			// Reapply the original ANSI sequence
			sb.WriteString(currentStyle)
		} else {
			// Not in selection, just copy the character
			sb.WriteString(char)
		}

		currentPos++
		i++
	}

	return sb.String()
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

// rgbTo16ColorFg converts a hex color to the closest 16-color foreground ANSI code
func rgbTo16ColorFg(hexColor string) int {
	return rgbTo16ColorBase(hexColor, 30) // Foreground colors start at 30
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

// renderLeftColumn formats the left side of a side-by-side diff
func renderLeftColumn(fileName string, dl *DiffLine, colWidth int) string {
	t := theme.CurrentTheme()

	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(t.DiffContextBg())
		return contextLineStyle.Width(colWidth).Render("")
	}

	removedLineStyle, _, contextLineStyle, lineNumberStyle := createStyles(t)

	// Determine line style based on line type
	var marker string
	var bgStyle lipgloss.Style
	switch dl.Kind {
	case LineRemoved:
		marker = removedLineStyle.Foreground(t.DiffRemoved()).Render("-")
		bgStyle = removedLineStyle
		lineNumberStyle = lineNumberStyle.Foreground(t.DiffRemoved()).Background(t.DiffRemovedLineNumberBg())
	case LineAdded:
		marker = "?"
		bgStyle = contextLineStyle
	case LineContext:
		marker = contextLineStyle.Render(" ")
		bgStyle = contextLineStyle
	}

	// Format line number
	lineNum := ""
	if dl.OldLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.OldLineNo)
	}

	// Create the line prefix
	prefix := lineNumberStyle.Render(lineNum + " " + marker)

	// Apply syntax highlighting
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	// Apply intra-line highlighting for removed lines
	if dl.Kind == LineRemoved && len(dl.Segments) > 0 {
		content = applyHighlighting(content, dl.Segments, LineRemoved, t.DiffHighlightRemoved())
	}

	// Add a padding space for removed lines
	if dl.Kind == LineRemoved {
		content = bgStyle.Render(" ") + content
	}

	// Create the final line and truncate if needed
	lineText := prefix + content
	return bgStyle.MaxHeight(1).Width(colWidth).Render(
		ansi.Truncate(
			lineText,
			colWidth,
			lipgloss.NewStyle().Background(bgStyle.GetBackground()).Foreground(t.TextMuted()).Render("..."),
		),
	)
}

// renderRightColumn formats the right side of a side-by-side diff
func renderRightColumn(fileName string, dl *DiffLine, colWidth int) string {
	t := theme.CurrentTheme()

	if dl == nil {
		contextLineStyle := lipgloss.NewStyle().Background(t.DiffContextBg())
		return contextLineStyle.Width(colWidth).Render("")
	}

	_, addedLineStyle, contextLineStyle, lineNumberStyle := createStyles(t)

	// Determine line style based on line type
	var marker string
	var bgStyle lipgloss.Style
	switch dl.Kind {
	case LineAdded:
		marker = addedLineStyle.Foreground(t.DiffAdded()).Render("+")
		bgStyle = addedLineStyle
		lineNumberStyle = lineNumberStyle.Foreground(t.DiffAdded()).Background(t.DiffAddedLineNumberBg())
	case LineRemoved:
		marker = "?"
		bgStyle = contextLineStyle
	case LineContext:
		marker = contextLineStyle.Render(" ")
		bgStyle = contextLineStyle
	}

	// Format line number
	lineNum := ""
	if dl.NewLineNo > 0 {
		lineNum = fmt.Sprintf("%6d", dl.NewLineNo)
	}

	// Create the line prefix
	prefix := lineNumberStyle.Render(lineNum + " " + marker)

	// Apply syntax highlighting
	content := highlightLine(fileName, dl.Content, bgStyle.GetBackground())

	// Apply intra-line highlighting for added lines
	if dl.Kind == LineAdded && len(dl.Segments) > 0 {
		content = applyHighlighting(content, dl.Segments, LineAdded, t.DiffHighlightAdded())
	}

	// Add a padding space for added lines
	if dl.Kind == LineAdded {
		content = bgStyle.Render(" ") + content
	}

	// Create the final line and truncate if needed
	lineText := prefix + content
	return bgStyle.MaxHeight(1).Width(colWidth).Render(
		ansi.Truncate(
			lineText,
			colWidth,
			lipgloss.NewStyle().Background(bgStyle.GetBackground()).Foreground(t.TextMuted()).Render("..."),
		),
	)
}

// -------------------------------------------------------------------------
// Public API
// -------------------------------------------------------------------------

// RenderSideBySideHunk formats a hunk for side-by-side display
func RenderSideBySideHunk(fileName string, h Hunk, opts ...SideBySideOption) string {
	// Apply options to create the configuration
	config := NewSideBySideConfig(opts...)

	// Make a copy of the hunk so we don't modify the original
	hunkCopy := Hunk{Lines: make([]DiffLine, len(h.Lines))}
	copy(hunkCopy.Lines, h.Lines)

	// Highlight changes within lines
	HighlightIntralineChanges(&hunkCopy)

	// Pair lines for side-by-side display
	pairs := pairLines(hunkCopy.Lines)

	// Calculate column width
	colWidth := config.TotalWidth / 2

	leftWidth := colWidth
	rightWidth := config.TotalWidth - colWidth
	var sb strings.Builder
	for _, p := range pairs {
		leftStr := renderLeftColumn(fileName, p.left, leftWidth)
		rightStr := renderRightColumn(fileName, p.right, rightWidth)
		sb.WriteString(leftStr + rightStr + "\n")
	}

	return sb.String()
}

// FormatDiff creates a side-by-side formatted view of a diff
func FormatDiff(diffText string, opts ...SideBySideOption) (string, error) {
	diffResult, err := ParseUnifiedDiff(diffText)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, h := range diffResult.Hunks {
		sb.WriteString(RenderSideBySideHunk(diffResult.OldFile, h, opts...))
	}

	return sb.String(), nil
}

// GenerateDiff creates a unified diff from two file contents
func GenerateDiff(beforeContent, afterContent, fileName string) (string, int, int) {
	// remove the cwd prefix and ensure consistent path format
	// this prevents issues with absolute paths in different environments
	cwd := config.WorkingDirectory()
	fileName = strings.TrimPrefix(fileName, cwd)
	fileName = strings.TrimPrefix(fileName, "/")

	var (
		unified   = udiff.Unified("a/"+fileName, "b/"+fileName, beforeContent, afterContent)
		additions = 0
		removals  = 0
	)

	lines := strings.SplitSeq(unified, "\n")
	for line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removals++
		}
	}

	return unified, additions, removals
}
