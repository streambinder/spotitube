package cui

const (
	// PromptExit is the identifier for exiting prompt
	PromptExit = 1 << iota
	// PromptBinary is the identifier for prompt with binary input
	PromptBinary
	// PromptInput is the identifier for prompt with textual input
	PromptInput
	_
	// ClearAppend is the identifier for pre-append panel clearing
	ClearAppend
	// ErrorAppend is the identifier for error mode append
	ErrorAppend
	// WarningAppend is the identifier for warning mode append
	WarningAppend
	// DebugAppend is the identifier for debug mode append
	DebugAppend
	_
	// PanelLeftTop is the identifier for panel at left-top
	PanelLeftTop
	// PanelLeftBottom is the identifier for panel at left-bottom
	PanelLeftBottom
	// PanelRight is the identifier for panel at right
	PanelRight
	_ProgressBar
	_Prompt
	_
	// OrientationLeft is the identifier for text left orientation
	OrientationLeft
	// OrientationCenter is the identifier for text center orientation
	OrientationCenter
	// OrientationRight is the identifier for text right orientation
	OrientationRight
	_
	// ColorBlack is the identifier for text black color
	ColorBlack
	// ColorRed is the identifier for text red color
	ColorRed
	// ColorGreen is the identifier for text green color
	ColorGreen
	// ColorYellow is the identifier for text yellow color
	ColorYellow
	// ColorBlue is the identifier for text blue color
	ColorBlue
	// ColorMagenta is the identifier for text magenta color
	ColorMagenta
	// ColorCyan is the identifier for text cyan color
	ColorCyan
	// ColorWhite is the identifier for text white color
	ColorWhite
	_
	// StyleBold is the identifier for text bold style
	StyleBold
	// StyleItalic is the identifier for text italic style
	StyleItalic
	// StyleUnderline is the identifier for text underline style
	StyleUnderline
	_
	// ParagraphAutoReturn is the identifier for text autoreturning paragraph format, to fit words in lines
	ParagraphAutoReturn
	_
	// LogEnable is the identifier for log writing flag
	LogEnable
	_
	// GuiBareMode is the identifier for make gui as bare as possible
	GuiBareMode
	// GuiDebugMode is the identifier for enabling debug mode
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
