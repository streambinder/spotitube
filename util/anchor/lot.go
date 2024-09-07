package anchor

import (
	"fmt"

	"atomicgo.dev/cursor"
	"github.com/fatih/color"
	"github.com/streambinder/spotitube/util"
)

const idle = "idle"

var idleColor = color.New(color.FgWhite)

type Lot struct {
	anchor
	id    int
	alias string
	style *color.Color
}

func formatAlias(alias string) string {
	return fmt.Sprintf("(%s) ", alias)
}

func (lot *Lot) Print(message string) {
	lot.window.lock.Lock()
	defer lot.window.lock.Unlock()
	defer cursor.Bottom()

	lot.data = message
	lot.window.up(len(lot.window.lots))
	for _, lot := range lot.window.lots {
		lot.write()
		lot.window.down()
	}
}

func (lot *Lot) Printf(format string, a ...any) {
	lot.Print(fmt.Sprintf(format, a...))
}

func (lot *Lot) Wipe() {
	lot.Print(idle)
}

func (lot *Lot) Close(messages ...string) {
	lot.style = color.New(color.FgWhite)
	lot.Print(util.First(messages, "done"))
}

func (lot *Lot) write() {
	dataStyle := lot.style
	if lot.data == idle {
		dataStyle = idleColor
	}
	fmt.Print(lot.style.Sprint(formatAlias(lot.alias)), dataStyle.Sprint(lot.data))
}
