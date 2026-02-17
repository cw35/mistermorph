package slackcmd

import (
	"context"

	slackruntime "github.com/quailyquaily/mistermorph/internal/channelruntime/slack"
)

// RunOptions configures the reusable Slack runtime entrypoint.
type RunOptions = slackruntime.RunOptions

// Run starts slack runtime with explicit options.
func Run(ctx context.Context, d Dependencies, opts RunOptions) error {
	return slackruntime.Run(ctx, slackruntime.Dependencies(d), slackruntime.RunOptions(opts))
}
