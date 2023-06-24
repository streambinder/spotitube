package anchor

import (
	"fmt"
	"sync"

	"atomicgo.dev/cursor"
	"github.com/pterm/pterm"
)

const (
	Black        = pterm.FgBlack
	Blue         = pterm.FgBlue
	Cyan         = pterm.FgCyan
	Gray         = pterm.FgGray
	Green        = pterm.FgGreen
	LightBlue    = pterm.FgLightBlue
	LightCyan    = pterm.FgLightCyan
	LightGreen   = pterm.FgLightGreen
	LightMagenta = pterm.FgLightMagenta
	LightRed     = pterm.FgLightRed
	LightWhite   = pterm.FgLightWhite
	LightYellow  = pterm.FgLightYellow
	Magenta      = pterm.FgMagenta
	Normal       = pterm.FgDefault
	Red          = pterm.FgRed
	White        = pterm.FgWhite
	Yellow       = pterm.FgYellow
	_
	cursorAnchor = -iota
	cursorDefault
)

type Color pterm.Color

type window struct {
	anchors     []*anchor
	lots        []*lot
	aliases     map[string]int
	anchorColor pterm.Color
	lock        sync.RWMutex
}

type anchor struct {
	data   string
	window *window
}

func Window(anchorColors ...pterm.Color) *window {
	color := Normal
	if len(anchorColors) > 0 {
		color = pterm.Color(anchorColors[0])
	}
	return &window{
		anchors:     []*anchor{},
		lots:        []*lot{},
		aliases:     make(map[string]int),
		anchorColor: color,
		lock:        sync.RWMutex{},
	}
}

func (window *window) Lot(alias string) *lot {
	window.lock.Lock()
	defer window.lock.Unlock()

	if id, ok := window.aliases[alias]; ok {
		return window.lots[id]
	}

	lot := &lot{
		anchor: anchor{
			data:   "",
			window: window,
		},
		id:    len(window.lots),
		alias: alias,
		style: pterm.Bold,
	}
	window.aliases[alias] = len(window.lots)
	window.lots = append(window.lots, lot)
	fmt.Println()
	return lot
}

func (window *window) Printf(format string, a ...any) {
	window.print(false, fmt.Sprintf(format, a...))
}

func (window *window) AnchorPrintf(format string, a ...any) {
	window.print(true, pterm.FgRed.Sprintf(format, a...))
}

func (window *window) up(lines ...int) {
	amount := 1
	if len(lines) > 0 {
		amount = lines[0]
	}
	cursor.UpAndClear(amount)
	cursor.StartOfLine()
}

func (window *window) down() {
	cursor.DownAndClear(1)
	cursor.StartOfLine()
}

func (window *window) shift(lines int) {
	if lines <= 0 && lines != cursorAnchor && lines != cursorDefault {
		return
	}

	fmt.Println()
	window.up()

	if lines == cursorAnchor {
		lines = len(window.lots)
	} else if lines == cursorDefault {
		lines = len(window.lots) + len(window.anchors)
	}

	for i := 0; i < lines; i++ {
		if i < len(window.lots) {
			window.lots[len(window.lots)-1-i].write()
		} else {
			i := i - len(window.lots)
			fmt.Print(window.anchors[len(window.anchors)-1-i].data)
		}
		window.up()
	}
}

func (window *window) print(doAnchor bool, data string) {
	window.lock.Lock()
	defer window.lock.Unlock()
	defer cursor.Bottom()

	if doAnchor {
		window.anchors = append(window.anchors, &anchor{data, window})
		window.shift(cursorAnchor)
	} else {
		window.shift(cursorDefault)
	}

	fmt.Print(data)
}
