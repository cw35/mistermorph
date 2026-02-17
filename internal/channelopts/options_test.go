package channelopts

import (
	"testing"
	"time"
)

func TestParseTelegramAllowedChatIDs(t *testing.T) {
	got, err := ParseTelegramAllowedChatIDs([]string{" 1 ", "", "-100", "1"})
	if err != nil {
		t.Fatalf("ParseTelegramAllowedChatIDs() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (%#v)", len(got), got)
	}
	if got[0] != 1 || got[1] != -100 {
		t.Fatalf("got = %#v, want [1 -100]", got)
	}
}

func TestParseTelegramAllowedChatIDsInvalid(t *testing.T) {
	if _, err := ParseTelegramAllowedChatIDs([]string{"abc"}); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestBuildTelegramRunOptionsTaskTimeoutFallback(t *testing.T) {
	opts, err := BuildTelegramRunOptions(
		TelegramConfig{
			AllowedChatIDsRaw:                    []string{"100"},
			TaskTimeout:                          0,
			GlobalTaskTimeout:                    2 * time.Minute,
			PollTimeout:                          30 * time.Second,
			MaxConcurrency:                       3,
			DefaultGroupTriggerMode:              "smart",
			DefaultAddressingConfidenceThreshold: 0.6,
			DefaultAddressingInterjectThreshold:  0.6,
		},
		TelegramInput{
			BotToken:    "token",
			TaskTimeout: 0,
		},
	)
	if err != nil {
		t.Fatalf("BuildTelegramRunOptions() error = %v", err)
	}
	if opts.TaskTimeout != 2*time.Minute {
		t.Fatalf("task timeout = %v, want 2m", opts.TaskTimeout)
	}
	if len(opts.AllowedChatIDs) != 1 || opts.AllowedChatIDs[0] != 100 {
		t.Fatalf("allowed chat ids = %#v, want [100]", opts.AllowedChatIDs)
	}
}

func TestBuildSlackRunOptionsTaskTimeoutFallback(t *testing.T) {
	opts := BuildSlackRunOptions(
		SlackConfig{
			TaskTimeout:                          0,
			GlobalTaskTimeout:                    3 * time.Minute,
			MaxConcurrency:                       3,
			DefaultGroupTriggerMode:              "smart",
			DefaultAddressingConfidenceThreshold: 0.6,
			DefaultAddressingInterjectThreshold:  0.6,
		},
		SlackInput{
			BotToken:    "xoxb-1",
			AppToken:    "xapp-1",
			TaskTimeout: 0,
		},
	)
	if opts.TaskTimeout != 3*time.Minute {
		t.Fatalf("task timeout = %v, want 3m", opts.TaskTimeout)
	}
}
