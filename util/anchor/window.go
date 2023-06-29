package anchor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"atomicgo.dev/cursor"
	"github.com/fatih/color"
	"github.com/streambinder/spotitube/util"
)

const (
	Black   = color.FgBlack
	Blue    = color.FgBlue
	Cyan    = color.FgCyan
	Green   = color.FgGreen
	Magenta = color.FgMagenta
	Normal  = color.Reset
	Red     = color.FgRed
	White   = color.FgWhite
	Yellow  = color.FgYellow
	_
	cursorAnchor = -iota
	cursorDefault
)

type Color color.Attribute

type window struct {
	anchors     []*anchor
	lots        []*lot
	aliases     map[string]int
	anchorColor *color.Color
	lock        sync.RWMutex
}

type anchor struct {
	data   string
	window *window
}

func Window(anchorColors ...color.Attribute) *window {
	return &window{
		anchors:     []*anchor{},
		lots:        []*lot{},
		aliases:     make(map[string]int),
		anchorColor: color.New(util.First(anchorColors, Normal)),
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
		style: color.New(color.Bold),
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
	window.print(true, window.anchorColor.Sprintf(format, a...))
}

func (window *window) up(lines ...int) {
	cursor.UpAndClear(util.First(lines, 1))
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

func (window *window) Reads(label string, a ...interface{}) (value string) {
	window.lock.Lock()
	defer window.lock.Unlock()
	defer cursor.Bottom()
	window.shift(cursorDefault)
	fmt.Printf(label+" ", a...)
	value, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")
	return
}
