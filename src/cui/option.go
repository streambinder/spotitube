package cui

const (
	// PromptExit : identifier for exiting prompt
	PromptExit = 1 << iota
	// PromptBinary : identifier for prompt with binary input
	PromptBinary
	// PromptInput : identifier for prompt with textual input
	PromptInput
	_
	// ClearAppend : identifier for pre-append panel clearing
	ClearAppend
	// ErrorAppend : identifier for error mode append
	ErrorAppend
	// WarningAppend : identifier for warning mode append
	WarningAppend
	// DebugAppend : identifier for debug mode append
	DebugAppend
	_
	// PanelLeftTop : identifier for panel at left-top
	PanelLeftTop
	// PanelLeftBottom : identifier for panel at left-bottom
	PanelLeftBottom
	// PanelRight : identifier for panel at right
	PanelRight
	_ProgressBar
	_Prompt
	_
	// OrientationLeft : identifier for text left orientation
	OrientationLeft
	// OrientationCenter : identifier for text center orientation
	OrientationCenter
	// OrientationRight : identifier for text right orientation
	OrientationRight
	_
	// ColorBlack : identifier for text black color
	ColorBlack
	// ColorRed : identifier for text red color
	ColorRed
	// ColorGreen : identifier for text green color
	ColorGreen
	// ColorYellow : identifier for text yellow color
	ColorYellow
	// ColorBlue : identifier for text blue color
	ColorBlue
	// ColorMagenta : identifier for text magenta color
	ColorMagenta
	// ColorCyan : identifier for text cyan color
	ColorCyan
	// ColorWhite : identifier for text white color
	ColorWhite
	_
	// StyleBold : identifier for text bold style
	StyleBold
	// StyleItalic : identifier for text italic style
	StyleItalic
	// StyleUnderline : identifier for text underline style
	StyleUnderline
	_
	// ParagraphAutoReturn : identifier for text autoreturning paragraph format, to fit words in lines
	ParagraphAutoReturn
	_
	// LogEnable : identifier for log writing flag
	LogEnable
	_
	// GuiBareMode : identifier for make gui as bare as possible
	GuiBareMode
	// GuiDebugMode : identifier for enabling debug mode
	GuiDebugMode
)

// Options is an alias to uint64
type Options = uint64

// Option is an alias to Options, used only for readability purposes
type Option = Options

func hasOption(options Options, option Option) bool {
	return (options & option) != 0
}

func (c *CUI) hasOption(option Option) bool {
	return hasOption(c.Options, option)
}

func (operation Operation) hasOption(option Option) bool {
	return hasOption(operation.Options, option)
}
