package idempotency

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func ManualContactKey(contactID string) string {
	return fmt.Sprintf("manual:%s:%s", token(contactID), uuid.NewString())
}

func ProactiveShareKey(contactID string, itemID string, sourceChatID int64, sourceChatType string) string {
	return fmt.Sprintf(
		"proactive:%s:%s:%d:%s",
		token(contactID),
		token(itemID),
		sourceChatID,
		token(sourceChatType),
	)
}

func MessageEnvelopeKey(messageID string) string {
	return "msg:" + token(messageID)
}

func token(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
