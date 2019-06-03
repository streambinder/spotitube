package cui

import (
	"fmt"
	"math"
	"strings"

	"github.com/jroimartin/gocui"
)

// ProgressHalfIncrease : increase progress bar by half-step
func (c *CUI) ProgressHalfIncrease() error {
	view, err := c.view(_ProgressBar)
	if err != nil {
		return err
	}
	width, _ := view.Size()
	return c.progressIncrease(int(math.Floor(c.ProgressOffset))*100/c.ProgressMax, int(math.Floor(c.ProgressOffset))*width/c.ProgressMax, 0.5)
}

// ProgressIncrease : increase progress bar
func (c *CUI) ProgressIncrease() error {
	view, err := c.view(_ProgressBar)
	if err != nil {
		return err
	}
	width, _ := view.Size()
	return c.progressIncrease(int(math.Floor(c.ProgressOffset))*100/c.ProgressMax, int(math.Floor(c.ProgressOffset))*width/c.ProgressMax, 1)
}

// ProgressFill : fill up the progress bar
func (c *CUI) ProgressFill() error {
	view, err := c.view(_ProgressBar)
	if err != nil {
		return err
	}
	width, _ := view.Size()
	return c.progressIncrease(100, width, float64(c.ProgressMax))
}

func (c *CUI) progressIncrease(percentage int, progressIncrease int, increaseAmount float64) error {
	if c.hasOption(GuiBareMode) {
		return nil
	}

	c.Update(func(gui *gocui.Gui) error {
		view, err := c.view(_ProgressBar)
		if err != nil {
			return err
		}
		view.Clear()
		view.Title = fmt.Sprintf(" %d %% ", percentage)
		fmt.Fprint(view, strings.Repeat(c.ProgressSprint, progressIncrease))
		c.ProgressOffset += increaseAmount
		return nil
	})
	return nil

}
