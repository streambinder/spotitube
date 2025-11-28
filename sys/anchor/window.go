package anchor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"atomicgo.dev/cursor"
	"github.com/fatih/color"
	"github.com/streambinder/spotitube/sys"
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

type Window struct {
	anchors        []*anchor
	lots           []*Lot
	aliases        map[string]int
	anchorColor    *color.Color
	lotHeaderColor *color.Color
	lock           sync.RWMutex
	plain          bool
}

type anchor struct {
	data   string
	window *Window
}

func New(anchorColors ...color.Attribute) *Window {
	return &Window{
		anchors:        []*anchor{},
		lots:           []*Lot{},
		aliases:        make(map[string]int),
		anchorColor:    color.New(sys.First(anchorColors, Normal)),
		lotHeaderColor: color.New(color.Bold),
		lock:           sync.RWMutex{},
		plain:          false,
	}
}

func (window *Window) EnablePlainMode() {
	window.anchorColor = color.New(Normal)
	window.lotHeaderColor = color.New(Normal)
	window.plain = true
}

func (window *Window) Lot(alias string) *Lot {
	window.lock.Lock()
	defer window.lock.Unlock()

	if id, ok := window.aliases[alias]; ok {
		return window.lots[id]
	}

	lot := &Lot{
		anchor: anchor{
			data:   "",
			window: window,
		},
		id:    len(window.lots),
		alias: alias,
		style: window.lotHeaderColor,
	}
	window.aliases[alias] = len(window.lots)
	window.lots = append(window.lots, lot)

	if !lot.window.plain {
		fmt.Println()
	}
	return lot
}

func (window *Window) Printf(format string, a ...any) {
	window.print(false, fmt.Sprintf(format, a...))
}

func (window *Window) AnchorPrintf(format string, a ...any) {
	window.print(true, window.anchorColor.Sprintf(format, a...))
}

func (window *Window) up(lines ...int) {
	cursor.UpAndClear(sys.First(lines, 1))
	cursor.StartOfLine()
}

func (window *Window) down() {
	cursor.DownAndClear(1)
	cursor.StartOfLine()
}

func (window *Window) shift(lines int) {
	if lines <= 0 && lines != cursorAnchor && lines != cursorDefault {
		return
	}

	fmt.Println()
	window.up()

	switch lines {
	case cursorAnchor:
		lines = len(window.lots)
	case cursorDefault:
		lines = len(window.lots) + len(window.anchors)
	}

	for i := 0; i < lines; i++ {
		if i < len(window.lots) {
			lotID := len(window.lots) - 1 - i
			window.lots[lotID].write()
		} else {
			anchorIndex := i - len(window.lots)
			anchorID := len(window.anchors) - 1 - anchorIndex
			fmt.Print(window.anchors[anchorID].data)
		}
		window.up()
	}
}

func (window *Window) print(doAnchor bool, data string) {
	window.lock.Lock()
	defer window.lock.Unlock()
	defer cursor.Bottom()

	if window.plain {
		fmt.Println(data)
		return
	}

	if doAnchor {
		window.anchors = append(window.anchors, &anchor{data, window})
		window.shift(cursorAnchor)
	} else {
		window.shift(cursorDefault)
	}
	fmt.Print(data)
}

func (window *Window) Reads(label string, a ...interface{}) (value string) {
	window.lock.Lock()
	defer window.lock.Unlock()
	defer cursor.Bottom()

	if !window.plain {
		window.shift(cursorDefault)
	}

	fmt.Printf(label+" ", a...)
	value = sys.ErrWrap("")(bufio.NewReader(os.Stdin).ReadString('\n'))
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\n")
	value = strings.Trim(value, "\r")

	// Clear the blank line created by Enter, then move up to the prompt line without clearing it
	if !window.plain {
		cursor.ClearLine()
		cursor.Up(1)
		cursor.StartOfLine()
	}

	return value
}
