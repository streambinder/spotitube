package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/streambinder/spotitube/sys"
)

func init() {
	cmdRoot.AddCommand(cmdDaemon())
}

// cmdDaemon runs the sync pipeline in a loop, with idle time between cycles;
// it reuses the sync flags and pipeline, but never prompts (no --manual)
// and always uses plain line-oriented output
func cmdDaemon() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "daemon",
		Short:        "Run sync in a loop, with idle time between cycles",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, params, err := prepareRun(cmd)
			if err != nil {
				return err
			}
			interval := sys.ErrWrap(15 * time.Minute)(cmd.Flags().GetDuration("interval"))
			if interval <= 0 {
				return errors.New("--interval must be positive")
			}
			// a daemon never prompts and never anchors a TUI across cycles:
			// quiet logs and plain output are implied, not optional
			params.quiet = true
			tui.EnablePlainMode()
			return runDaemon(ctx, params, interval)
		},
		PreRun: func(cmd *cobra.Command, _ []string) {
			preRunSync(cmd)
		},
	}
	addSyncFlags(cmd)
	cmd.Flags().Duration("interval", 15*time.Minute, "Idle time between sync cycles")
	return cmd
}

// runDaemon repeats sync cycles in a loop: cycles run strictly sequentially
// (a new cycle starts only after the previous one finished) with interval of
// idle time in between; SIGINT/SIGTERM stop the loop once the in-flight cycle
// winds down — workers check for cancellation between phases, but an
// in-flight network request runs to completion first; a dead Spotify session
// is fatal so the process exits and the service manager can restart it visibly
func runDaemon(ctx context.Context, params syncParams, interval time.Duration) error {
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	tui.Printf("daemon: syncing every %s of idle time", interval)

	first := true
	for cycle := 1; ; cycle++ {
		setupPipeline()
		stats, err := runSyncCycle(sigCtx, params, first)
		if err != nil {
			if errors.Is(err, errSpotifyAuthDead) {
				return err
			}
			tui.AnchorPrintf("daemon: cycle %d failed: %s", cycle, err)
		} else {
			tui.Printf("daemon: cycle complete — %d synced, %d skipped, %d collisions ignored, took %s",
				stats.synced, stats.skipped, stats.collisions, stats.took)
		}
		first = false

		select {
		case <-sigCtx.Done():
			tui.Printf("daemon: shutting down")
			return nil
		case <-time.After(interval):
		}
	}
}
