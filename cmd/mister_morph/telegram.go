package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quailyquaily/mister_morph/agent"
	"github.com/quailyquaily/mister_morph/llm"
	"github.com/quailyquaily/mister_morph/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type telegramJob struct {
	ChatID  int64
	Text    string
	Version uint64
}

type telegramChatWorker struct {
	Jobs    chan telegramJob
	Version uint64
}

func newTelegramCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Run a Telegram bot that chats with the agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := strings.TrimSpace(viper.GetString("telegram.bot_token"))
			if token == "" {
				return fmt.Errorf("missing telegram.bot_token (set via --telegram-bot-token or MISTER_MORPH_TELEGRAM_BOT_TOKEN)")
			}

			baseURL := strings.TrimRight(strings.TrimSpace(viper.GetString("telegram.base_url")), "/")
			if baseURL == "" {
				baseURL = "https://api.telegram.org"
			}

			allowed := make(map[int64]bool)
			for _, s := range viper.GetStringSlice("telegram.allowed_chat_ids") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				id, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid telegram.allowed_chat_ids entry %q: %w", s, err)
				}
				allowed[id] = true
			}

			logger, err := loggerFromViper()
			if err != nil {
				return err
			}
			slog.SetDefault(logger)

			client, err := llmClientFromConfig()
			if err != nil {
				return err
			}
			model := viper.GetString("model")
			reg := registryFromViper()
			logOpts := logOptionsFromViper()

			cfg := agent.Config{
				MaxSteps:       viper.GetInt("max_steps"),
				ParseRetries:   viper.GetInt("parse_retries"),
				MaxTokenBudget: viper.GetInt("max_token_budget"),
				PlanMode:       viper.GetString("plan.mode"),
			}

			pollTimeout := viper.GetDuration("telegram.poll_timeout")
			if pollTimeout <= 0 {
				pollTimeout = 30 * time.Second
			}
			taskTimeout := viper.GetDuration("telegram.task_timeout")
			if taskTimeout <= 0 {
				taskTimeout = viper.GetDuration("timeout")
			}
			if taskTimeout <= 0 {
				taskTimeout = 10 * time.Minute
			}
			maxConc := viper.GetInt("telegram.max_concurrency")
			if maxConc <= 0 {
				maxConc = 3
			}
			sem := make(chan struct{}, maxConc)

			historyMax := viper.GetInt("telegram.history_max_messages")
			if historyMax <= 0 {
				historyMax = 20
			}

			httpClient := &http.Client{Timeout: 60 * time.Second}
			api := newTelegramAPI(httpClient, baseURL, token)

			me, err := api.getMe(context.Background())
			if err != nil {
				return err
			}

			botUser := me.Username
			botID := me.ID
			aliases := viper.GetStringSlice("telegram.aliases")
			for i := range aliases {
				aliases[i] = strings.TrimSpace(aliases[i])
			}

			var (
				mu      sync.Mutex
				history = make(map[int64][]llm.Message)
				workers = make(map[int64]*telegramChatWorker)
				offset  int64
			)

			logger.Info("telegram_start",
				"base_url", baseURL,
				"bot_username", botUser,
				"bot_id", botID,
				"poll_timeout", pollTimeout.String(),
				"task_timeout", taskTimeout.String(),
				"max_concurrency", maxConc,
				"history_max_messages", historyMax,
			)

			getOrStartWorkerLocked := func(chatID int64) *telegramChatWorker {
				if w, ok := workers[chatID]; ok && w != nil {
					return w
				}
				w := &telegramChatWorker{Jobs: make(chan telegramJob, 16)}
				workers[chatID] = w

				go func(chatID int64, w *telegramChatWorker) {
					for job := range w.Jobs {
						// Global concurrency limit.
						sem <- struct{}{}
						func() {
							defer func() { <-sem }()

							mu.Lock()
							h := append([]llm.Message(nil), history[chatID]...)
							curVersion := w.Version
							mu.Unlock()

							// If there was a /reset after this job was queued, drop history for this run.
							if job.Version != curVersion {
								h = nil
							}

							_ = api.sendChatAction(context.Background(), chatID, "typing")

							ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
							final, _, runErr := runTelegramTask(ctx, logger, logOpts, client, reg, cfg, job.Text, model, h)
							cancel()

							if runErr != nil {
								_ = api.sendMessage(context.Background(), chatID, "error: "+runErr.Error(), true)
								return
							}

							outText := formatFinalOutput(final)
							if err := api.sendMessageChunked(context.Background(), chatID, outText); err != nil {
								logger.Warn("telegram_send_error", "error", err.Error())
							}

							mu.Lock()
							// Respect resets that happened while the task was running.
							if w.Version != curVersion {
								history[chatID] = nil
							}
							cur := history[chatID]
							cur = append(cur,
								llm.Message{Role: "user", Content: job.Text},
								llm.Message{Role: "assistant", Content: outText},
							)
							if len(cur) > historyMax {
								cur = cur[len(cur)-historyMax:]
							}
							history[chatID] = cur
							mu.Unlock()
						}()
					}
				}(chatID, w)

				return w
			}

			for {
				updates, nextOffset, err := api.getUpdates(context.Background(), offset, pollTimeout)
				if err != nil {
					logger.Warn("telegram_get_updates_error", "error", err.Error())
					time.Sleep(1 * time.Second)
					continue
				}
				offset = nextOffset

				for _, u := range updates {
					msg := u.Message
					if msg == nil {
						msg = u.EditedMessage
					}
					if msg == nil {
						msg = u.ChannelPost
					}
					if msg == nil {
						msg = u.EditedChannelPost
					}
					if msg == nil || msg.Chat == nil {
						continue
					}
					chatID := msg.Chat.ID
					text := strings.TrimSpace(msg.Text)
					if text == "" {
						continue
					}
					if len(allowed) > 0 && !allowed[chatID] {
						logger.Warn("telegram_unauthorized_chat", "chat_id", chatID)
						_ = api.sendMessage(context.Background(), chatID, "unauthorized", true)
						continue
					}

					chatType := strings.ToLower(strings.TrimSpace(msg.Chat.Type))
					isGroup := chatType == "group" || chatType == "supergroup"

					cmdWord, cmdArgs := splitCommand(text)
					switch normalizeSlashCommand(cmdWord) {
					case "/start", "/help":
						help := "Send a message and I will run it as an agent task.\n" +
							"Commands: /ask <task>, /reset, /id\n\n" +
							"Group chats: use /ask <task>, reply to me, or mention @" + botUser + ".\n" +
							"Note: if Bot Privacy Mode is enabled, I may not receive normal group messages (so aliases won't trigger unless I receive the message)."
						_ = api.sendMessage(context.Background(), chatID, help, true)
						continue
					case "/reset":
						mu.Lock()
						delete(history, chatID)
						if w := getOrStartWorkerLocked(chatID); w != nil {
							w.Version++
						}
						mu.Unlock()
						_ = api.sendMessage(context.Background(), chatID, "ok (reset)", true)
						continue
					case "/id":
						_ = api.sendMessage(context.Background(), chatID, fmt.Sprintf("chat_id=%d type=%s", chatID, chatType), true)
						continue
					case "/ask":
						if strings.TrimSpace(cmdArgs) == "" {
							_ = api.sendMessage(context.Background(), chatID, "usage: /ask <task>", true)
							continue
						}
						text = strings.TrimSpace(cmdArgs)
					default:
						if isGroup {
							trigger, ok := groupTriggerReason(msg, botUser, botID, aliases)
							if !ok {
								logger.Debug("telegram_group_ignored",
									"chat_id", chatID,
									"type", chatType,
									"text_len", len(text),
								)
								continue
							}
							logger.Info("telegram_group_trigger",
								"chat_id", chatID,
								"type", chatType,
								"trigger", trigger,
							)
							text = stripBotMentions(text, botUser)
							if strings.TrimSpace(text) == "" {
								_ = api.sendMessage(context.Background(), chatID, "usage: /ask <task> (or send text with a mention/reply)", true)
								continue
							}
						}
					}

					// Enqueue to per-chat worker (per chat serial; across chats parallel).
					mu.Lock()
					w := getOrStartWorkerLocked(chatID)
					v := w.Version
					mu.Unlock()
					logger.Info("telegram_task_enqueued", "chat_id", chatID, "type", chatType, "text_len", len(text))
					w.Jobs <- telegramJob{ChatID: chatID, Text: text, Version: v}
				}
			}
		},
	}

	cmd.Flags().String("telegram-bot-token", "", "Telegram bot token.")
	cmd.Flags().String("telegram-base-url", "https://api.telegram.org", "Telegram API base URL.")
	cmd.Flags().StringArray("telegram-allowed-chat-id", nil, "Allowed chat id(s). If empty, allows all.")
	cmd.Flags().StringArray("telegram-alias", nil, "Bot alias keywords (group messages containing these may trigger a response).")
	cmd.Flags().Duration("telegram-poll-timeout", 30*time.Second, "Long polling timeout for getUpdates.")
	cmd.Flags().Duration("telegram-task-timeout", 0, "Per-message agent timeout (0 uses --timeout).")
	cmd.Flags().Int("telegram-max-concurrency", 3, "Max number of chats processed concurrently.")
	cmd.Flags().Int("telegram-history-max-messages", 20, "Max chat history messages to keep per chat.")

	_ = viper.BindPFlag("telegram.bot_token", cmd.Flags().Lookup("telegram-bot-token"))
	_ = viper.BindPFlag("telegram.base_url", cmd.Flags().Lookup("telegram-base-url"))
	_ = viper.BindPFlag("telegram.allowed_chat_ids", cmd.Flags().Lookup("telegram-allowed-chat-id"))
	_ = viper.BindPFlag("telegram.aliases", cmd.Flags().Lookup("telegram-alias"))
	_ = viper.BindPFlag("telegram.poll_timeout", cmd.Flags().Lookup("telegram-poll-timeout"))
	_ = viper.BindPFlag("telegram.task_timeout", cmd.Flags().Lookup("telegram-task-timeout"))
	_ = viper.BindPFlag("telegram.max_concurrency", cmd.Flags().Lookup("telegram-max-concurrency"))
	_ = viper.BindPFlag("telegram.history_max_messages", cmd.Flags().Lookup("telegram-history-max-messages"))

	viper.SetDefault("telegram.base_url", "https://api.telegram.org")
	viper.SetDefault("telegram.poll_timeout", 30*time.Second)
	viper.SetDefault("telegram.history_max_messages", 20)
	viper.SetDefault("telegram.aliases", []string{})
	viper.SetDefault("telegram.max_concurrency", 3)

	return cmd
}

func runTelegramTask(ctx context.Context, logger *slog.Logger, logOpts agent.LogOptions, client llm.Client, reg *tools.Registry, cfg agent.Config, task string, model string, history []llm.Message) (*agent.Final, *agent.Context, error) {
	if reg == nil {
		reg = registryFromViper()
	}
	promptSpec, err := promptSpecWithSkills(ctx, logger, logOpts, task, client, model)
	if err != nil {
		return nil, nil, err
	}
	engine := agent.New(
		client,
		reg,
		cfg,
		promptSpec,
		agent.WithLogger(logger),
		agent.WithLogOptions(logOpts),
	)
	return engine.Run(ctx, task, agent.RunOptions{Model: model, History: history})
}

func formatFinalOutput(final *agent.Final) string {
	if final == nil {
		return ""
	}
	switch v := final.Output.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		b, _ := json.MarshalIndent(v, "", "  ")
		return strings.TrimSpace(string(b))
	}
}

// Telegram API

type telegramAPI struct {
	http    *http.Client
	baseURL string
	token   string
}

func newTelegramAPI(httpClient *http.Client, baseURL, token string) *telegramAPI {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &telegramAPI{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
	}
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message,omitempty"`
	// Some clients/users may @mention by editing an existing message.
	EditedMessage     *telegramMessage `json:"edited_message,omitempty"`
	ChannelPost       *telegramMessage `json:"channel_post,omitempty"`
	EditedChannelPost *telegramMessage `json:"edited_channel_post,omitempty"`
}

type telegramMessage struct {
	MessageID int64            `json:"message_id"`
	Chat      *telegramChat    `json:"chat,omitempty"`
	From      *telegramUser    `json:"from,omitempty"`
	ReplyTo   *telegramMessage `json:"reply_to_message,omitempty"`
	Entities  []telegramEntity `json:"entities,omitempty"`
	Text      string           `json:"text,omitempty"`
}

type telegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type,omitempty"` // private|group|supergroup|channel
}

type telegramUser struct {
	ID       int64  `json:"id"`
	IsBot    bool   `json:"is_bot,omitempty"`
	Username string `json:"username,omitempty"`
}

type telegramEntity struct {
	Type   string        `json:"type"`
	Offset int           `json:"offset"`
	Length int           `json:"length"`
	User   *telegramUser `json:"user,omitempty"` // for text_mention
}

type telegramGetUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

type telegramGetMeResponse struct {
	OK     bool         `json:"ok"`
	Result telegramUser `json:"result"`
}

func (api *telegramAPI) getMe(ctx context.Context) (*telegramUser, error) {
	url := fmt.Sprintf("%s/bot%s/getMe", api.baseURL, api.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := api.http.Do(req)
	if err != nil {
		return nil, err
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("telegram http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out telegramGetMeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram getMe: ok=false")
	}
	return &out.Result, nil
}

func (api *telegramAPI) getUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]telegramUpdate, int64, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	url := fmt.Sprintf("%s/bot%s/getUpdates?timeout=%d", api.baseURL, api.token, secs)
	if offset > 0 {
		url += fmt.Sprintf("&offset=%d", offset)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout+5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, offset, err
	}
	resp, err := api.http.Do(req)
	if err != nil {
		return nil, offset, err
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, offset, fmt.Errorf("telegram http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out telegramGetUpdatesResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, offset, err
	}
	if !out.OK {
		return nil, offset, fmt.Errorf("telegram getUpdates: ok=false")
	}

	next := offset
	for _, u := range out.Result {
		if u.UpdateID >= next {
			next = u.UpdateID + 1
		}
	}
	return out.Result, next, nil
}

type telegramSendMessageRequest struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type telegramSendChatActionRequest struct {
	ChatID int64  `json:"chat_id"`
	Action string `json:"action"`
}

type telegramOKResponse struct {
	OK bool `json:"ok"`
}

func (api *telegramAPI) sendMessage(ctx context.Context, chatID int64, text string, disablePreview bool) error {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "(empty)"
	}
	reqBody := telegramSendMessageRequest{
		ChatID:                chatID,
		Text:                  text,
		DisableWebPagePreview: disablePreview,
	}
	b, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/bot%s/sendMessage", api.baseURL, api.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := api.http.Do(req)
	if err != nil {
		return err
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var ok telegramOKResponse
	_ = json.Unmarshal(raw, &ok)
	if !ok.OK {
		return fmt.Errorf("telegram sendMessage: ok=false")
	}
	return nil
}

func (api *telegramAPI) sendMessageChunked(ctx context.Context, chatID int64, text string) error {
	const max = 3500
	text = strings.TrimSpace(text)
	if text == "" {
		return api.sendMessage(ctx, chatID, "(empty)", true)
	}
	for len(text) > 0 {
		chunk := text
		if len(chunk) > max {
			chunk = chunk[:max]
		}
		if err := api.sendMessage(ctx, chatID, chunk, true); err != nil {
			return err
		}
		text = strings.TrimSpace(text[len(chunk):])
	}
	return nil
}

func splitCommand(text string) (cmd string, rest string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	i := strings.IndexAny(text, " \n\t")
	if i == -1 {
		return text, ""
	}
	return text[:i], strings.TrimSpace(text[i:])
}

func normalizeSlashCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || !strings.HasPrefix(cmd, "/") {
		return ""
	}
	// Allow "/cmd@BotName" variants by stripping "@...".
	if at := strings.IndexByte(cmd, '@'); at >= 0 {
		cmd = cmd[:at]
	}
	return strings.ToLower(cmd)
}

func groupTriggerReason(msg *telegramMessage, botUser string, botID int64, aliases []string) (string, bool) {
	if msg == nil {
		return "", false
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return "", false
	}

	// Reply-to-bot.
	if msg.ReplyTo != nil && msg.ReplyTo.From != nil && msg.ReplyTo.From.ID == botID {
		return "reply", true
	}

	// Entity-based mention of the bot (text_mention includes user id).
	for _, e := range msg.Entities {
		if strings.ToLower(strings.TrimSpace(e.Type)) == "text_mention" && e.User != nil && e.User.ID == botID {
			return "text_mention", true
		}
	}

	// Explicit @mention.
	if botUser != "" && strings.Contains(strings.ToLower(text), "@"+strings.ToLower(botUser)) {
		return "at_mention", true
	}

	// Alias keywords.
	lower := strings.ToLower(text)
	for _, a := range aliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if strings.Contains(lower, a) {
			return "alias:" + a, true
		}
	}
	return "", false
}

func stripBotMentions(text, botUser string) string {
	text = strings.TrimSpace(text)
	if text == "" || botUser == "" {
		return text
	}
	mention := "@" + botUser
	// Remove common mention patterns (case-insensitive).
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(mention))
	if idx >= 0 {
		text = strings.TrimSpace(text[:idx] + text[idx+len(mention):])
	}
	return strings.TrimSpace(text)
}

func (api *telegramAPI) sendChatAction(ctx context.Context, chatID int64, action string) error {
	action = strings.TrimSpace(action)
	if action == "" {
		action = "typing"
	}
	reqBody := telegramSendChatActionRequest{ChatID: chatID, Action: action}
	b, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s/bot%s/sendChatAction", api.baseURL, api.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := api.http.Do(req)
	if err != nil {
		return err
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
