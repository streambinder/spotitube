package cui

import (
	"strings"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

// Font applies font styling defined via given fontConst
// over given message
func Font(message string, fontConst uint64) string {
	return applyFont(message, map[uint64]color.Attribute{
		StyleBold:      color.Bold,
		StyleItalic:    color.Italic,
		StyleUnderline: color.Underline,
	}[fontConst])
}

func applyFont(message string, fontAttribute color.Attribute) string {
	return color.New(fontAttribute).Sprintf(message)
}

func styleFont(message string, options Option) string {
	if hasOption(options, StyleBold) {
		return Font(message, StyleBold)
	} else if hasOption(options, StyleItalic) {
		return Font(message, StyleItalic)
	} else if hasOption(options, StyleUnderline) {
		return Font(message, StyleUnderline)
	}
	return message
}

// Color applies colorization defined via given colorConst
// over given message
func Color(message string, colorConst uint64) string {
	return applyColor(message, map[uint64]color.Attribute{
		ColorBlack:   color.FgBlack,
		ColorRed:     color.FgRed,
		ColorGreen:   color.FgGreen,
		ColorYellow:  color.FgYellow,
		ColorBlue:    color.FgBlue,
		ColorMagenta: color.FgMagenta,
		ColorCyan:    color.FgCyan,
		ColorWhite:   color.FgWhite,
	}[colorConst])
}

func applyColor(message string, colorAttribute color.Attribute) string {
	return color.New(colorAttribute).Sprintf(message)
}

func styleColor(message string, options Options) string {
	if hasOption(options, ColorBlack) {
		return Color(message, ColorBlack)
	} else if hasOption(options, ColorRed) {
		return Color(message, ColorRed)
	} else if hasOption(options, ColorGreen) {
		return Color(message, ColorGreen)
	} else if hasOption(options, ColorYellow) {
		return Color(message, ColorYellow)
	} else if hasOption(options, ColorBlue) {
		return Color(message, ColorBlue)
	} else if hasOption(options, ColorMagenta) {
		return Color(message, ColorMagenta)
	} else if hasOption(options, ColorCyan) {
		return Color(message, ColorCyan)
	} else if hasOption(options, ColorWhite) {
		return Color(message, ColorWhite)
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
