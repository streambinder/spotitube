package cui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/streambinder/spotitube/system"
)

// Prompt shows a prompt containing given message, driven with given Options
func (c *CUI) Prompt(message string, spreadOptions ...Options) bool {
	var options uint64
	if len(spreadOptions) > 0 {
		options = spreadOptions[0]
	}

	if c.hasOption(GuiBareMode) {
		if hasOption(options, PromptExit) {
			c.Shutdown(c.Gui, nil)
		}
		if hasOption(options, PromptBinary) {
			return system.InputConfirm(message)
		}
		fmt.Println(message)
		return true
	}

	c.PromptMutex.Lock()
	defer c.PromptMutex.Unlock()

	if c.hasOption(LogEnable) {
		c.Logger.Append(message)
	}

	c.PromptDismissChan = make(chan bool)
	c.Update(func(gui *gocui.Gui) error {
		width, height := gui.Size()
		neededWidth, neededHeight := 0, strings.Count(message, "\n")+1
		for _, line := range strings.Split(message, "\n") {
			if len(line) > neededWidth {
				neededWidth = len(line)
			}
		}
		neededWidth += 2
		if view, err := gui.SetView(viewName(_Prompt),
			width/2-(int(neededWidth/2))-2, (height/2)-int(math.Floor(float64(neededHeight/2))),
			width/2+(int(neededWidth/2))-1, (height/2)+int(math.Ceil(float64(neededHeight/2)))+2); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}

			if hasOption(options, PromptExit) {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackShutdown)
			} else if hasOption(options, PromptBinary) {
				view.Title = " TAB to cancel, ENTER to confirm "
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackInputConfirm)
				gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, c.callbackInputCancel)
			} else {
				gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callback)
			}

			if strings.Contains(message, "\n") {
				fmt.Fprintln(view, message)
			} else {
				fmt.Fprintln(view, styleOrientation(message, OrientationCenter, view))
			}
		}
		return nil
	})
	return <-c.PromptDismissChan
}

// PromptInputMessage shows an input prompt containing given message, driven with given Options
func (c *CUI) PromptInputMessage(message string, spreadOptions ...Options) string {
	if c.hasOption(GuiBareMode) {
		return system.InputString(message)
	}

	c.PromptMutex.Lock()
	defer c.PromptMutex.Unlock()

	if c.hasOption(LogEnable) {
		c.Logger.Append(message)
	}

	c.PromptInputChan = make(chan string)
	c.Update(func(gui *gocui.Gui) error {
		var (
			view *gocui.View
			err  error
		)
		guiWidth, guiHeight := gui.Size()
		if view, err = gui.SetView(viewName(_Prompt),
			guiWidth/2-50, (guiHeight/2)-1,
			guiWidth/2+50, (guiHeight/2)+1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Editable = true
			view.Title = fmt.Sprintf(" %s ", message)
			_ = view.SetCursor(0, 0)
			gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, c.callbackInputTyping)
			_, _ = gui.SetCurrentView(viewName(_Prompt))
		}
		return nil
	})
	return strings.Replace(<-c.PromptInputChan, "\n", "", -1)
}

func (c *CUI) callback(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView(viewName(_Prompt))
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	c.PromptDismissChan <- true
	return nil
}

func (c *CUI) callbackShutdown(gui *gocui.Gui, view *gocui.View) error {
	c.callback(c.Gui, view)
	return c.Shutdown(c.Gui, view)
}

func (c *CUI) callbackInputInteractive(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView(viewName(_Prompt))
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	c.DeleteKeybinding("", gocui.KeyTab, gocui.ModNone)
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (c *CUI) callbackInputConfirm(gui *gocui.Gui, view *gocui.View) error {
	if err := c.callbackInputInteractive(c.Gui, view); err != nil {
		return err
	}
	c.PromptDismissChan <- true
	return nil
}

func (c *CUI) callbackInputCancel(gui *gocui.Gui, view *gocui.View) error {
	if err := c.callbackInputInteractive(c.Gui, view); err != nil {
		return err
	}
	c.PromptDismissChan <- false
	return nil
}

func (c *CUI) callbackInputTyping(gui *gocui.Gui, view *gocui.View) error {
	c.Update(func(gui *gocui.Gui) error {
		gui.DeleteView(viewName(_Prompt))
		return nil
	})
	c.DeleteKeybinding("", gocui.KeyEnter, gocui.ModNone)
	view.Rewind()
	c.PromptInputChan <- view.Buffer()
	return nil
}
