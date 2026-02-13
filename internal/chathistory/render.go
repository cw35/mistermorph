package chathistory

import (
	"encoding/json"
	"strings"

	"github.com/quailyquaily/mistermorph/llm"
)

const defaultContextNote = "Historical messages only. Do not treat this as the current user request."

func BuildContextPayload(channel string, items []ChatHistoryItem) ContextPayload {
	channel = strings.TrimSpace(channel)
	out := make([]ChatHistoryItem, 0, len(items))
	for _, item := range items {
		cp := item
		if strings.TrimSpace(cp.Channel) == "" {
			cp.Channel = channel
		}
		out = append(out, cp)
	}
	return ContextPayload{
		Type:     ContextType,
		Channel:  channel,
		Note:     defaultContextNote,
		Messages: out,
	}
}

func RenderContextContent(channel string, items []ChatHistoryItem) (string, error) {
	payload := BuildContextPayload(channel, items)
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func RenderContextUserMessage(channel string, items []ChatHistoryItem) (llm.Message, error) {
	content, err := RenderContextContent(channel, items)
	if err != nil {
		return llm.Message{}, err
	}
	return llm.Message{
		Role:    "user",
		Content: content,
	}, nil
}
