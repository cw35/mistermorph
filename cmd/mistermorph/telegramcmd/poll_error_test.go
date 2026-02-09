package telegramcmd

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestIsTelegramPollTimeoutError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
		{name: "wrapped deadline exceeded", err: fmt.Errorf("poll failed: %w", context.DeadlineExceeded), want: true},
		{name: "net timeout", err: timeoutErr{}, want: true},
		{name: "string deadline exceeded", err: errors.New("Get ...: context deadline exceeded"), want: true},
		{name: "non-timeout", err: errors.New("telegram http 401: unauthorized"), want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isTelegramPollTimeoutError(tc.err)
			if got != tc.want {
				t.Fatalf("isTelegramPollTimeoutError() = %v, want %v", got, tc.want)
			}
		})
	}
}
