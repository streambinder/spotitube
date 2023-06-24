package anchor

import (
	"fmt"

	"atomicgo.dev/cursor"
	"github.com/pterm/pterm"
)

type lot struct {
	anchor
	id    int
	alias string
	style pterm.Color
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
	lot.Printf("")
}

func (lot *lot) Close(messages ...string) {
	message := "done"
	if len(messages) > 0 {
		message = messages[0]
	}
	lot.style = pterm.FgDarkGray
	lot.Printf(message)
}

func (lot *lot) write() {
	fmt.Print(lot.style.Sprint(formatAlias(lot.alias), lot.data))
}
