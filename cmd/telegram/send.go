package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	telegramruntime "github.com/quailyquaily/mistermorph/internal/channelruntime/telegram"
	"github.com/spf13/cobra"
)

const (
	defaultTelegramBaseURL  = "https://api.telegram.org"
	defaultTelegramChunkLen = 3500
)

type sendOptions struct {
	ChatID     int64
	FilePath   string
	BaseURL    string
	HTTPClient *http.Client
}

func newSendCmd() *cobra.Command {
	var chatID int64
	var baseURL string

	cmd := &cobra.Command{
		Use:   "send <file-path>",
		Short: "Send file contents to a Telegram chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sent, err := runSend(cmd.Context(), sendOptions{
				ChatID:   chatID,
				FilePath: args[0],
				BaseURL:  baseURL,
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sent %d message chunk(s) to chat_id=%d\n", sent, chatID)
			return nil
		},
	}

	cmd.Flags().Int64Var(&chatID, "chat-id", 0, "Target Telegram chat id (required).")
	cmd.Flags().StringVar(&baseURL, "base-url", defaultTelegramBaseURL, "Telegram API base URL.")
	_ = cmd.MarkFlagRequired("chat-id")
	return cmd
}

func runSend(ctx context.Context, opts sendOptions) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.ChatID == 0 {
		return 0, fmt.Errorf("chat id is required")
	}
	token := telegramTokenFromEnv()
	if token == "" {
		return 0, fmt.Errorf("missing telegram token: set MISTER_MORPH_TELEGRAM_BOT_TOKEN (or TELEGRAM_BOT_TOKEN)")
	}
	filePath := strings.TrimSpace(opts.FilePath)
	if filePath == "" {
		return 0, fmt.Errorf("file path is required")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	text := string(raw)
	if text == "" {
		text = "(empty file)"
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = defaultTelegramBaseURL
	}
	chunks := splitTelegramTextChunks(text, defaultTelegramChunkLen)
	for i, chunk := range chunks {
		if err := telegramruntime.SendMessageHTML(ctx, client, baseURL, token, opts.ChatID, chunk, true); err != nil {
			return 0, fmt.Errorf("send chunk %d/%d: %w", i+1, len(chunks), err)
		}
	}
	return len(chunks), nil
}

func telegramTokenFromEnv() string {
	for _, key := range []string{"MISTER_MORPH_TELEGRAM_BOT_TOKEN", "TELEGRAM_BOT_TOKEN"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func splitTelegramTextChunks(text string, maxChunkLen int) []string {
	if maxChunkLen <= 0 {
		maxChunkLen = defaultTelegramChunkLen
	}
	if text == "" {
		return []string{"(empty file)"}
	}
	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+maxChunkLen-1)/maxChunkLen)
	for len(runes) > 0 {
		n := maxChunkLen
		if len(runes) < n {
			n = len(runes)
		}
		chunks = append(chunks, string(runes[:n]))
		runes = runes[n:]
	}
	return chunks
}
