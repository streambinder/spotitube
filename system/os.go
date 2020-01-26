package system

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/0xAX/notificator"
)

// Notify provides a simple function interface to the notification firing logic
func Notify(app string, appIcon string, title string, content string) error {
	if err := notificator.New(notificator.Options{
		AppName:     app,
		DefaultIcon: appIcon,
	}).Push(title, content, "", notificator.UR_NORMAL); err != nil {
		return err
	}

	return nil
}

// Which returns true if command is found, false otherwise
func Which(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// TrapSignal provides a simple signal subscription flow
func TrapSignal(s os.Signal, f func()) {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, s)
	go func() {
		for range channel {
			f()
		}
	}()
}
