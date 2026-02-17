package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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

type slackAddressingLLMOutput struct {
	Addressed      bool                         `json:"addressed"`
	Confidence     float64                      `json:"confidence"`
	WannaInterject *slackWannaInterjectDecision `json:"wanna_interject,omitempty"`
	Interject      float64                      `json:"interject"`
	Impulse        float64                      `json:"impulse"`
	Reason         string                       `json:"reason"`
}

type slackWannaInterjectDecision bool

func (w *slackWannaInterjectDecision) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*w = false
		return nil
	}

	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*w = slackWannaInterjectDecision(b)
		return nil
	}

	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*w = slackWannaInterjectDecision(f >= 0.5)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		normalized := strings.ToLower(strings.TrimSpace(s))
		switch normalized {
		case "true":
			*w = true
			return nil
		case "false":
			*w = false
			return nil
		}
		fv, parseErr := strconv.ParseFloat(normalized, 64)
		if parseErr != nil {
			return fmt.Errorf("unsupported wanna_interject value: %q", s)
		}
		*w = slackWannaInterjectDecision(fv >= 0.5)
		return nil
	}

	return fmt.Errorf("unsupported wanna_interject json: %s", raw)
}

func quoteReplyThreadTSForGroupTrigger(event slackInboundEvent, dec slackGroupTriggerDecision) string {
	threadTS := strings.TrimSpace(event.ThreadTS)
	if threadTS != "" {
		return threadTS
	}
	if dec.Addressing.Impulse > 0.8 {
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
		ConfidenceThreshold:      addressingConfidenceThreshold,
		InterjectThreshold:       addressingInterjectThreshold,
		ExplicitReason:           explicitReason,
		ExplicitMatched:          explicitMentioned,
		AddressingFallbackReason: mode,
		AddressingTimeout:        addressingLLMTimeout,
		Addressing: func(addrCtx context.Context) (grouptrigger.Addressing, bool, error) {
			return slackAddressingDecisionViaLLM(addrCtx, client, model, event, history)
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

func slackAddressingDecisionViaLLM(ctx context.Context, client llm.Client, model string, event slackInboundEvent, history []chathistory.ChatHistoryItem) (grouptrigger.Addressing, bool, error) {
	if ctx == nil || client == nil {
		return grouptrigger.Addressing{}, false, nil
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return grouptrigger.Addressing{}, false, fmt.Errorf("missing model for addressing_llm")
	}
	personaIdentity := loadAddressingPersonaIdentity()
	if personaIdentity == "" {
		personaIdentity = "You are MisterMorph, a general-purpose AI agent that can use tools to complete tasks."
	}
	historyMessages := chathistory.BuildMessages(chathistory.ChannelSlack, history)
	systemPrompt := strings.TrimSpace(strings.Join([]string{
		personaIdentity,
		"You are deciding whether the latest Slack group message should trigger an agent run.",
		"Return strict JSON with fields: addressed (bool), confidence (0..1), wanna_interject (bool), interject (0..1), impulse (0..1), reason (string).",
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
		return grouptrigger.Addressing{}, false, err
	}
	raw := strings.TrimSpace(res.Text)
	if raw == "" {
		return grouptrigger.Addressing{}, false, fmt.Errorf("empty addressing_llm response")
	}
	var out slackAddressingLLMOutput
	if err := jsonutil.DecodeWithFallback(raw, &out); err != nil {
		return grouptrigger.Addressing{}, false, fmt.Errorf("invalid addressing_llm json")
	}
	wannaInterject := out.Interject > 0
	if out.WannaInterject != nil {
		wannaInterject = bool(*out.WannaInterject)
	}
	addressing := grouptrigger.Addressing{
		Addressed:      out.Addressed,
		Confidence:     clampAddressing01(out.Confidence),
		WannaInterject: wannaInterject,
		Interject:      clampAddressing01(out.Interject),
		Impulse:        clampAddressing01(out.Impulse),
		Reason:         strings.TrimSpace(out.Reason),
	}
	return addressing, true, nil
}

func clampAddressing01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func loadAddressingPersonaIdentity() string {
	spec := agent.PromptSpec{}
	promptprofile.ApplyPersonaIdentity(&spec, slog.Default())
	return strings.TrimSpace(spec.Identity)
}
