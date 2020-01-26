package cui

import (
	"strings"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

func applyFont(message string, fontAttribute color.Attribute) string {
	return color.New(fontAttribute).Sprintf(message)
}

// Font applies font styling defined via given fontConst
// over given message
func Font(message string, options Option) string {
	if hasOption(options, StyleBold) {
		return applyFont(message, color.Bold)
	} else if hasOption(options, StyleItalic) {
		return applyFont(message, color.Italic)
	} else if hasOption(options, StyleUnderline) {
		return applyFont(message, color.Underline)
	}
	return message
}

func applyColor(message string, colorAttribute color.Attribute) string {
	return color.New(colorAttribute).Sprintf(message)
}

// Color applies colorization defined via given colorConst
// over given message
func Color(message string, options Options) string {
	if hasOption(options, ColorBlack) {
		return applyColor(message, color.FgBlack)
	} else if hasOption(options, ColorRed) {
		return applyColor(message, color.FgRed)
	} else if hasOption(options, ColorGreen) {
		return applyColor(message, color.FgGreen)
	} else if hasOption(options, ColorYellow) {
		return applyColor(message, color.FgYellow)
	} else if hasOption(options, ColorBlue) {
		return applyColor(message, color.FgBlue)
	} else if hasOption(options, ColorMagenta) {
		return applyColor(message, color.FgMagenta)
	} else if hasOption(options, ColorCyan) {
		return applyColor(message, color.FgCyan)
	} else if hasOption(options, ColorWhite) {
		return applyColor(message, color.FgWhite)
	}
	return message
}

func styleOrientation(message string, options Option, view *gocui.View) string {
	var orientation uint64
	if hasOption(options, OrientationLeft) {
		orientation = OrientationLeft
	} else if hasOption(options, OrientationCenter) {
		orientation = OrientationCenter
	} else if hasOption(options, OrientationRight) {
		orientation = OrientationRight
	}

	var messageLines []string
	var lineLength, _ = view.Size()
	for _, line := range strings.Split(message, "\n") {
		if len(line) < lineLength {
			lineSpacing := (lineLength - len(line)) / 2
			if orientation == OrientationCenter {
				line = strings.Repeat(" ", lineSpacing) +
					line + strings.Repeat(" ", lineSpacing)
			} else if orientation == OrientationRight {
				line = strings.Repeat(" ", lineSpacing*2-1) + line
			}
		}
		messageLines = append(messageLines, line)
	}
	return strings.Join(messageLines, "\n")
}

func styleParagraph(message string, options Option, view *gocui.View) string {
	if hasOption(options, ParagraphAutoReturn) {
		var (
			messageParagraph string
			lineLength, _    = view.Size()
		)
		for len(message) > 0 {
			if len(message) < lineLength {
				messageParagraph = messageParagraph + message
				message = ""
			} else {
				messageParagraph = messageParagraph + message[:lineLength] + "\n"
				message = message[lineLength:]
			}
		}
		return messageParagraph
	}
	return message
}
