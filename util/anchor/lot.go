package anchor

import (
	"fmt"

	"atomicgo.dev/cursor"
	"github.com/fatih/color"
	"github.com/streambinder/spotitube/util"
)

const idle = "idle"

var idleColor = color.New(color.FgWhite)

type lot struct {
	anchor
	id    int
	alias string
	style *color.Color
}

func formatAlias(alias string) string {
	return fmt.Sprintf("(%s) ", alias)
}

func (lot *lot) Printf(format string, a ...any) {
	lot.window.lock.Lock()
	defer lot.window.lock.Unlock()
	defer cursor.Bottom()

	lot.data = fmt.Sprintf(format, a...)
	lot.window.up(len(lot.window.lots))
	for _, lot := range lot.window.lots {
		lot.write()
		lot.window.down()
	}
}

func (lot *lot) Wipe() {
	lot.Printf(idle)
}

func (lot *lot) Close(messages ...string) {
	lot.style = color.New(color.FgWhite)
	lot.Printf(util.First(messages, "done"))
}

func (lot *lot) write() {
	dataStyle := lot.style
	if lot.data == idle {
		dataStyle = idleColor
	}
	fmt.Print(lot.style.Sprint(formatAlias(lot.alias)), dataStyle.Sprint(lot.data))
}
