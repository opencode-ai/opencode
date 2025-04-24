package styles

import (
	"github.com/charmbracelet/glamour/ansi"
)

const defaultMargin = 1

// Helper functions for style pointers
func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }

func MarkdownTheme(focused bool) ansi.StyleConfig {
	if !focused {
		return ASCIIStyleConfig
	} else {
		return DraculaStyleConfig
	}
}

const (
	defaultListIndent      = 2
	defaultListLevelIndent = 4
)

var ASCIIStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Color:           stringPtr(ColorBackgroundDim),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
		},
		Indent:      uintPtr(1),
		IndentToken: stringPtr("| "),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			IndentToken: stringPtr(BaseStyle.Render(" ")),
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
		},
		LevelIndent: defaultListLevelIndent,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			BlockSuffix:     "\n",
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "# ",
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
			Prefix:          "###### ",
		},
	},
	Strikethrough: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		BlockPrefix:     "~~",
		BlockSuffix:     "~~",
	},
	Emph: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		BlockPrefix:     "*",
		BlockSuffix:     "*",
	},
	Strong: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		BlockPrefix:     "**",
		BlockSuffix:     "**",
	},
	HorizontalRule: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		Format:          "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix:     "• ",
		BackgroundColor: stringPtr(ColorBackground),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix:     ". ",
		BackgroundColor: stringPtr(ColorBackground),
	},
	Task: ansi.StyleTask{
		Ticked:   "[x] ",
		Unticked: "[ ] ",
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	ImageText: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		Format:          "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockPrefix:     "`",
			BlockSuffix:     "`",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			Margin: uintPtr(defaultMargin),
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			IndentToken: stringPtr(BaseStyle.Render(" ")),
		},
		CenterSeparator: stringPtr("|"),
		ColumnSeparator: stringPtr("|"),
		RowSeparator:    stringPtr("-"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
		BlockPrefix:     "\n* ",
	},
}

var DraculaStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr(ColorForeground),
			BackgroundColor: stringPtr(ColorBackground),
		},
		Indent:      uintPtr(defaultMargin),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#f1fa8c"),
			Italic:          boolPtr(true),
			BackgroundColor: stringPtr(ColorBackground),
		},
		Indent:      uintPtr(defaultMargin),
		IndentToken: stringPtr(BaseStyle.Render(" ")),
	},
	Paragraph: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	List: ansi.StyleList{
		LevelIndent: defaultMargin,
		StyleBlock: ansi.StyleBlock{
			IndentToken: stringPtr(BaseStyle.Render(" ")),
			StylePrimitive: ansi.StylePrimitive{
				Color:           stringPtr(ColorForeground),
				BackgroundColor: stringPtr(ColorBackground),
			},
		},
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix:     "\n",
			Color:           stringPtr(ColorPrimary),
			Bold:            boolPtr(true),
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "# ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "## ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "### ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "#### ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "##### ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix:          "###### ",
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut:      boolPtr(true),
		BackgroundColor: stringPtr(ColorBackground),
	},
	Emph: ansi.StylePrimitive{
		Color:           stringPtr("#f1fa8c"),
		Italic:          boolPtr(true),
		BackgroundColor: stringPtr(ColorBackground),
	},
	Strong: ansi.StylePrimitive{
		Bold:            boolPtr(true),
		Color:           stringPtr(darkBlue),
		BackgroundColor: stringPtr(ColorBackground),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:           stringPtr("#6272A4"),
		Format:          "\n--------\n",
		BackgroundColor: stringPtr(ColorBackground),
	},
	Item: ansi.StylePrimitive{
		BlockPrefix:     "• ",
		BackgroundColor: stringPtr(ColorBackground),
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix:     ". ",
		Color:           stringPtr("#8be9fd"),
		BackgroundColor: stringPtr(ColorBackground),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{
			BackgroundColor: stringPtr(ColorBackground),
		},
		Ticked:   "[✓] ",
		Unticked: "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:           stringPtr("#8be9fd"),
		Underline:       boolPtr(true),
		BackgroundColor: stringPtr(ColorBackground),
	},
	LinkText: ansi.StylePrimitive{
		Color:           stringPtr("#ff79c6"),
		BackgroundColor: stringPtr(ColorBackground),
	},
	Image: ansi.StylePrimitive{
		Color:           stringPtr("#8be9fd"),
		Underline:       boolPtr(true),
		BackgroundColor: stringPtr(ColorBackground),
	},
	ImageText: ansi.StylePrimitive{
		Color:           stringPtr("#ff79c6"),
		Format:          "Image: {{.text}} →",
		BackgroundColor: stringPtr(ColorBackground),
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#50fa7b"),
			BackgroundColor: stringPtr(ColorBackground),
		},
	},
	Text: ansi.StylePrimitive{
		BackgroundColor: stringPtr(ColorBackground),
	},
	DefinitionList: ansi.StyleBlock{},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           stringPtr(darkBlue),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Margin: uintPtr(defaultMargin),
		},
		Chroma: &ansi.Chroma{
			NameOther: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			Literal: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameException: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			LiteralDate: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			Text: ansi.StylePrimitive{
				Color:           stringPtr(ColorForeground),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Error: ansi.StylePrimitive{
				Color:           stringPtr("#f8f8f2"),
				BackgroundColor: stringPtr("#ff5555"),
			},
			Comment: ansi.StylePrimitive{
				Color:           stringPtr("#6272A4"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			CommentPreproc: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Keyword: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			KeywordReserved: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			KeywordNamespace: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			KeywordType: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Operator: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Punctuation: ansi.StylePrimitive{
				Color:           stringPtr(ColorForeground),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Name: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameBuiltin: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameTag: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameAttribute: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameClass: ansi.StylePrimitive{
				Color:           stringPtr("#8be9fd"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameConstant: ansi.StylePrimitive{
				Color:           stringPtr("#bd93f9"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameDecorator: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			NameFunction: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			LiteralNumber: ansi.StylePrimitive{
				Color:           stringPtr("#6EEFC0"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			LiteralString: ansi.StylePrimitive{
				Color:           stringPtr("#f1fa8c"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			LiteralStringEscape: ansi.StylePrimitive{
				Color:           stringPtr("#ff79c6"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			GenericDeleted: ansi.StylePrimitive{
				Color:           stringPtr("#ff5555"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			GenericEmph: ansi.StylePrimitive{
				Color:           stringPtr("#f1fa8c"),
				Italic:          boolPtr(true),
				BackgroundColor: stringPtr(ColorBackground),
			},
			GenericInserted: ansi.StylePrimitive{
				Color:           stringPtr("#50fa7b"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			GenericStrong: ansi.StylePrimitive{
				Color:           stringPtr("#ffb86c"),
				Bold:            boolPtr(true),
				BackgroundColor: stringPtr(ColorBackground),
			},
			GenericSubheading: ansi.StylePrimitive{
				Color:           stringPtr("#bd93f9"),
				BackgroundColor: stringPtr(ColorBackground),
			},
			Background: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: stringPtr(ColorBackground),
			},
			IndentToken: stringPtr(BaseStyle.Render(" ")),
		},
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix:     "\n* ",
		BackgroundColor: stringPtr(ColorBackground),
	},
}
