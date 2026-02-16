package telegramcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/quailyquaily/mistermorph/internal/chathistory"
	"github.com/quailyquaily/mistermorph/internal/jsonutil"
	"github.com/quailyquaily/mistermorph/internal/llminspect"
	"github.com/quailyquaily/mistermorph/llm"
)

type telegramAddressingLLMDecision struct {
	Addressed  bool    `json:"addressed"`
	Confidence float64 `json:"confidence"`
	Interject  float64 `json:"interject"`
	Impulse    float64 `json:"impulse"`
	Reason     string  `json:"reason"`
}

func addressingDecisionViaLLM(
	ctx context.Context,
	client llm.Client,
	model string,
	msg *telegramMessage,
	text string,
	history []chathistory.ChatHistoryItem,
) (telegramAddressingLLMDecision, bool, error) {
	if ctx == nil || client == nil {
		return telegramAddressingLLMDecision{}, false, nil
	}
	text = strings.TrimSpace(text)
	model = strings.TrimSpace(model)
	if model == "" {
		return telegramAddressingLLMDecision{}, false, fmt.Errorf("missing model for addressing_llm")
	}

	historyMessages := chathistory.BuildMessages(chathistory.ChannelTelegram, history)
	currentMessage := telegramAddressingPromptCurrentMessage{
		Text: text,
	}
	if msg != nil {
		if msg.From != nil {
			currentMessage.Sender.ID = msg.From.ID
			currentMessage.Sender.IsBot = msg.From.IsBot
			currentMessage.Sender.Username = strings.TrimSpace(msg.From.Username)
			currentMessage.Sender.DisplayName = strings.TrimSpace(telegramDisplayName(msg.From))
		}
		if msg.Chat != nil {
			currentMessage.Sender.ChatID = msg.Chat.ID
			currentMessage.Sender.ChatType = strings.TrimSpace(msg.Chat.Type)
		}
	}
	sys, user, err := renderTelegramAddressingPrompts(currentMessage, historyMessages)
	if err != nil {
		return telegramAddressingLLMDecision{}, false, fmt.Errorf("render addressing prompts: %w", err)
	}

	res, err := client.Chat(llminspect.WithModelScene(ctx, "telegram.addressing_decision"), llm.Request{
		Model:     model,
		ForceJSON: true,
		Messages: []llm.Message{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return telegramAddressingLLMDecision{}, false, err
	}

	raw := strings.TrimSpace(res.Text)
	if raw == "" {
		return telegramAddressingLLMDecision{}, false, fmt.Errorf("empty addressing_llm response")
	}

	var out telegramAddressingLLMDecision
	if err := jsonutil.DecodeWithFallback(raw, &out); err != nil {
		return telegramAddressingLLMDecision{}, false, fmt.Errorf("invalid addressing_llm json")
	}

	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	if out.Impulse < 0 {
		out.Impulse = 0
	}
	if out.Impulse > 1 {
		out.Impulse = 1
	}
	if out.Interject < 0 {
		out.Interject = 0
	}
	if out.Interject > 1 {
		out.Interject = 1
	}
	out.Reason = strings.TrimSpace(out.Reason)
	return out, true, nil
}
