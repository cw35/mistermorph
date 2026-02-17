package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/quailyquaily/mistermorph/internal/chathistory"
	"github.com/quailyquaily/mistermorph/internal/grouptrigger"
	"github.com/quailyquaily/mistermorph/internal/jsonutil"
	"github.com/quailyquaily/mistermorph/internal/llminspect"
	"github.com/quailyquaily/mistermorph/internal/promptprofile"
	"github.com/quailyquaily/mistermorph/llm"
)

type slackGroupTriggerDecision = grouptrigger.Decision

type slackAddressingLLMDecision struct {
	Addressed  bool    `json:"addressed"`
	Confidence float64 `json:"confidence"`
	Interject  float64 `json:"interject"`
	Impulse    float64 `json:"impulse"`
	Reason     string  `json:"reason"`
}

func quoteReplyThreadTSForGroupTrigger(event slackInboundEvent, dec slackGroupTriggerDecision) string {
	threadTS := strings.TrimSpace(event.ThreadTS)
	if threadTS != "" {
		return threadTS
	}
	if dec.AddressingImpulse > 0.8 {
		return strings.TrimSpace(event.MessageTS)
	}
	return ""
}

func decideSlackGroupTrigger(
	ctx context.Context,
	client llm.Client,
	model string,
	event slackInboundEvent,
	botUserID string,
	mode string,
	addressingLLMTimeout time.Duration,
	addressingConfidenceThreshold float64,
	addressingInterjectThreshold float64,
	history []chathistory.ChatHistoryItem,
) (slackGroupTriggerDecision, bool, error) {
	explicitReason, explicitMentioned := slackExplicitMentionReason(event, botUserID)
	return grouptrigger.Decide(ctx, grouptrigger.DecideOptions{
		Mode:                     mode,
		DefaultMode:              "smart",
		ConfidenceThreshold:      addressingConfidenceThreshold,
		InterjectThreshold:       addressingInterjectThreshold,
		DefaultConfidence:        0.6,
		DefaultInterject:         0.6,
		ExplicitReason:           explicitReason,
		ExplicitMatched:          explicitMentioned,
		AddressingFallbackReason: mode,
		AddressingTimeout:        addressingLLMTimeout,
		Addressing: func(addrCtx context.Context) (grouptrigger.AddressingDecision, bool, error) {
			llmDec, llmOK, llmErr := slackAddressingDecisionViaLLM(addrCtx, client, model, event, history)
			if llmErr != nil {
				return grouptrigger.AddressingDecision{}, false, llmErr
			}
			return grouptrigger.AddressingDecision{
				Addressed:  llmDec.Addressed,
				Confidence: llmDec.Confidence,
				Interject:  llmDec.Interject,
				Impulse:    llmDec.Impulse,
				Reason:     strings.TrimSpace(llmDec.Reason),
			}, llmOK, nil
		},
	})
}

func slackExplicitMentionReason(event slackInboundEvent, botUserID string) (string, bool) {
	if event.IsAppMention {
		return "app_mention", true
	}
	if strings.TrimSpace(botUserID) != "" && strings.Contains(event.Text, "<@"+strings.TrimSpace(botUserID)+">") {
		return "mention", true
	}
	if event.IsThreadMessage {
		return "thread_reply", true
	}
	return "", false
}

func slackAddressingDecisionViaLLM(ctx context.Context, client llm.Client, model string, event slackInboundEvent, history []chathistory.ChatHistoryItem) (slackAddressingLLMDecision, bool, error) {
	if ctx == nil || client == nil {
		return slackAddressingLLMDecision{}, false, nil
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return slackAddressingLLMDecision{}, false, fmt.Errorf("missing model for addressing_llm")
	}
	personaIdentity := loadAddressingPersonaIdentity()
	if personaIdentity == "" {
		personaIdentity = "You are MisterMorph, a general-purpose AI agent that can use tools to complete tasks."
	}
	historyMessages := chathistory.BuildMessages(chathistory.ChannelSlack, history)
	systemPrompt := strings.TrimSpace(strings.Join([]string{
		personaIdentity,
		"You are deciding whether the latest Slack group message should trigger an agent run.",
		"Return strict JSON with fields: addressed (bool), confidence (0..1), interject (0..1), impulse (0..1), reason (string).",
		"`addressed=true` means the user is clearly asking the bot or directly addressing the bot in context.",
	}, "\n"))
	userPayload, _ := json.Marshal(map[string]any{
		"current_message": map[string]any{
			"team_id":       event.TeamID,
			"channel_id":    event.ChannelID,
			"chat_type":     event.ChatType,
			"message_ts":    event.MessageTS,
			"thread_ts":     event.ThreadTS,
			"user_id":       event.UserID,
			"text":          event.Text,
			"mention_users": append([]string(nil), event.MentionUsers...),
		},
		"chat_history_messages": historyMessages,
	})
	res, err := client.Chat(llminspect.WithModelScene(ctx, "slack.addressing_decision"), llm.Request{
		Model:     model,
		ForceJSON: true,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(userPayload)},
		},
	})
	if err != nil {
		return slackAddressingLLMDecision{}, false, err
	}
	raw := strings.TrimSpace(res.Text)
	if raw == "" {
		return slackAddressingLLMDecision{}, false, fmt.Errorf("empty addressing_llm response")
	}
	var out slackAddressingLLMDecision
	if err := jsonutil.DecodeWithFallback(raw, &out); err != nil {
		return slackAddressingLLMDecision{}, false, fmt.Errorf("invalid addressing_llm json")
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	if out.Interject < 0 {
		out.Interject = 0
	}
	if out.Interject > 1 {
		out.Interject = 1
	}
	if out.Impulse < 0 {
		out.Impulse = 0
	}
	if out.Impulse > 1 {
		out.Impulse = 1
	}
	out.Reason = strings.TrimSpace(out.Reason)
	return out, true, nil
}

func loadAddressingPersonaIdentity() string {
	spec := agent.PromptSpec{}
	promptprofile.ApplyPersonaIdentity(&spec, slog.Default())
	return strings.TrimSpace(spec.Identity)
}
