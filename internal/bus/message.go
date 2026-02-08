package bus

import (
	"fmt"
	"time"

	"github.com/quailyquaily/mistermorph/internal/channels"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type Channel string

const (
	ChannelTelegram Channel = Channel(channels.Telegram)
	ChannelSlack    Channel = Channel(channels.Slack)
	ChannelDiscord  Channel = Channel(channels.Discord)
	ChannelMAEP     Channel = Channel(channels.MAEP)
)

type MessageExtensions struct {
	PlatformMessageID string `json:"platform_message_id,omitempty"`
	ReplyTo           string `json:"reply_to,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
}

type BusMessage struct {
	ID              string            `json:"id"`
	Direction       Direction         `json:"direction"`
	Channel         Channel           `json:"channel"`
	Topic           string            `json:"topic"`
	ConversationKey string            `json:"conversation_key"`
	ParticipantKey  string            `json:"participant_key"`
	IdempotencyKey  string            `json:"idempotency_key"`
	CorrelationID   string            `json:"correlation_id"`
	CausationID     string            `json:"causation_id,omitempty"`
	PayloadBase64   string            `json:"payload_base64"`
	CreatedAt       time.Time         `json:"created_at"`
	Extensions      MessageExtensions `json:"extensions,omitempty"`
}

func (m BusMessage) Validate() error {
	if m.ID != "" {
		if err := validateOptionalCanonicalString("id", m.ID); err != nil {
			return err
		}
	}
	if m.Direction != "" {
		switch m.Direction {
		case DirectionInbound, DirectionOutbound:
		default:
			return fmt.Errorf("direction must be inbound|outbound")
		}
	}

	if m.Channel != "" {
		switch m.Channel {
		case ChannelTelegram, ChannelSlack, ChannelDiscord, ChannelMAEP:
		default:
			return fmt.Errorf("channel is invalid")
		}
	}

	if err := ValidateTopic(m.Topic); err != nil {
		return wrapError(CodeInvalidTopic, err)
	}
	if err := validateRequiredCanonicalString("conversation_key", m.ConversationKey); err != nil {
		return err
	}
	if m.ParticipantKey != "" {
		if err := validateOptionalCanonicalString("participant_key", m.ParticipantKey); err != nil {
			return err
		}
	}
	if err := validateRequiredCanonicalString("idempotency_key", m.IdempotencyKey); err != nil {
		return err
	}
	if m.CorrelationID != "" {
		if err := validateOptionalCanonicalString("correlation_id", m.CorrelationID); err != nil {
			return err
		}
	}
	if m.CausationID != "" {
		if err := validateOptionalCanonicalString("causation_id", m.CausationID); err != nil {
			return err
		}
	}

	if err := validateRequiredCanonicalString("payload_base64", m.PayloadBase64); err != nil {
		return err
	}
	if _, err := DecodeMessageEnvelope(m.Topic, m.PayloadBase64); err != nil {
		return err
	}
	if m.Extensions.PlatformMessageID != "" {
		if err := validateOptionalCanonicalString("extensions.platform_message_id", m.Extensions.PlatformMessageID); err != nil {
			return err
		}
	}
	if m.Extensions.ReplyTo != "" {
		if err := validateOptionalCanonicalString("extensions.reply_to", m.Extensions.ReplyTo); err != nil {
			return err
		}
	}
	if m.Extensions.SessionID != "" {
		if err := validateUUIDv7Field("extensions.session_id", m.Extensions.SessionID); err != nil {
			return err
		}
	}

	return nil
}

func (m BusMessage) Envelope() (MessageEnvelope, error) {
	return DecodeMessageEnvelope(m.Topic, m.PayloadBase64)
}
