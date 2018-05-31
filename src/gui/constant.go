package gui

const (
	// OptionNil : identifier for no option
	OptionNil = 1 << iota
	_
	// PromptDismissable : identifier dismissable prompt
	PromptDismissable
	// PromptDismissableWithExit : identifier dismissable with exiting prompt
	PromptDismissableWithExit
	_
	// PanelLeftTop : identifier for panel at left-top
	PanelLeftTop
	// PanelLeftBottom : identifier for panel at left-bottom
	PanelLeftBottom
	// PanelRight : identifier for panel at right
	PanelRight
	// PanelLoading : identifier for loading panel
	PanelLoading
	_
	// OrientationLeft : identifier for text left orientation
	OrientationLeft
	// OrientationCenter : identifier for text center orientation
	OrientationCenter
	// OrientationRight : identifier for text right orientation
	OrientationRight
	_
	// FontColorBlack : identifier for text black color
	FontColorBlack
	// FontColorRed : identifier for text red color
	FontColorRed
	// FontColorGreen : identifier for text green color
	FontColorGreen
	// FontColorYellow : identifier for text yellow color
	FontColorYellow
	// FontColorBlue : identifier for text blue color
	FontColorBlue
	// FontColorMagenta : identifier for text magenta color
	FontColorMagenta
	// FontColorCyan : identifier for text cyan color
	FontColorCyan
	// FontColorWhite : identifier for text white color
	FontColorWhite
	_
	// FontStyleBold : identifier for text bold style
	FontStyleBold
	// FontStyleUnderline : identifier for text underline style
	FontStyleUnderline
	// FontStyleReverse : identifier for text reverse style
	FontStyleReverse
	_
	// ParagraphStyleStandard : identifier for text standard paragraph format
	ParagraphStyleStandard
	// ParagraphStyleAutoReturn : identifier for text autoreturning paragraph format, to fit words in lines
	ParagraphStyleAutoReturn
	_
	// LogNoWrite : identifier for log writing temporarily disable (if Gui has a Logger)
	LogNoWrite
)
