package telegramcmd

import (
	"context"

	telegramruntime "github.com/quailyquaily/mistermorph/internal/channelruntime/telegram"
)

// RunOptions configures the reusable telegram runtime entrypoint.
type RunOptions = telegramruntime.RunOptions

// Run starts telegram runtime with explicit options.
func Run(ctx context.Context, d Dependencies, opts RunOptions) error {
	return telegramruntime.Run(ctx, telegramruntime.Dependencies(d), telegramruntime.RunOptions(opts))
}
