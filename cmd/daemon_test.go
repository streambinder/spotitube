package cmd

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/sys"
	"github.com/streambinder/spotitube/sys/anchor"
	"github.com/streambinder/spotitube/sys/cmd"
	"github.com/stretchr/testify/assert"
)

func TestRunDaemon(t *testing.T) {
	oldTUI := tui
	t.Cleanup(func() { tui = oldTUI })
	tui = anchor.New(anchor.Red)
	tui.EnablePlainMode()
	defer mockey.UnPatchAll()

	// already-cancelled context shuts down after the first cycle
	mockey.Mock(runSyncCycle).Return(cycleStats{}, nil).Build()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.NoError(t, runDaemon(ctx, syncParams{}, time.Minute))
	mockey.UnPatchAll()

	// dead session is fatal
	mockey.Mock(runSyncCycle).Return(cycleStats{}, errSpotifyAuthDead).Build()
	assert.ErrorIs(t, runDaemon(context.Background(), syncParams{}, time.Minute), errSpotifyAuthDead)
	mockey.UnPatchAll()

	// failing cycles are logged and retried until shutdown
	mockey.Mock(runSyncCycle).Return(cycleStats{}, errors.New("boom")).Build()
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	assert.NoError(t, runDaemon(ctx, syncParams{}, 10*time.Millisecond))
	mockey.UnPatchAll()

	// cycles run sequentially with idle time in between
	var calls atomic.Int32
	ctx, cancel = context.WithCancel(context.Background())
	mockey.Mock(runSyncCycle).To(func(_ context.Context, _ syncParams, _ bool) (cycleStats, error) {
		if calls.Add(1) == 2 {
			cancel()
		}
		return cycleStats{}, nil
	}).Build()
	assert.NoError(t, runDaemon(ctx, syncParams{}, 10*time.Millisecond))
	assert.Equal(t, int32(2), calls.Load())
}

func TestCmdDaemon(t *testing.T) {
	t.Cleanup(cleanup)
	defer mockey.UnPatchAll()

	// a failing environment check aborts before the daemon starts
	mockey.Mock(cmd.ValidateEnvironment).Return(errors.New("env boom")).Build()
	assert.ErrorContains(t, sys.ErrOnly(testExecute(cmdDaemon())), "env boom")
	mockey.UnPatchAll()

	mockey.Mock(cmd.ValidateEnvironment).Return(nil).Build()

	// daemon requires a positive interval
	assert.ErrorContains(t, sys.ErrOnly(testExecute(cmdDaemon(), "--interval", "0s")), "must be positive")

	// --manual is not a daemon flag: a daemon never prompts
	assert.ErrorContains(t, sys.ErrOnly(testExecute(cmdDaemon(), "--manual")), "unknown flag")

	// the daemon command delegates to the daemon loop with quiet logs,
	// so per-track fetch/skip lines stay out of the service logs
	mockey.Mock(runDaemon).To(func(_ context.Context, params syncParams, _ time.Duration) error {
		assert.True(t, params.quiet)
		return nil
	}).Build()
	assert.NoError(t, sys.ErrOnly(testExecute(cmdDaemon())))
}
