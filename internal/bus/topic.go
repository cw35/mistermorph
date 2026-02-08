package bus

import "fmt"

const (
	TopicShareProactiveV1 = "share.proactive.v1"
	TopicDMCheckinV1      = "dm.checkin.v1"
	TopicDMReplyV1        = "dm.reply.v1"
	TopicChatMessage      = "chat.message"
)

var topicSet = map[string]struct{}{
	TopicShareProactiveV1: {},
	TopicDMCheckinV1:      {},
	TopicDMReplyV1:        {},
	TopicChatMessage:      {},
}

func IsKnownTopic(topic string) bool {
	_, ok := topicSet[topic]
	return ok
}

func ValidateTopic(topic string) error {
	if err := validateRequiredCanonicalString("topic", topic); err != nil {
		return err
	}
	if !IsKnownTopic(topic) {
		return fmt.Errorf("topic is invalid")
	}
	return nil
}

func IsDialogueTopic(topic string) bool {
	switch topic {
	case TopicShareProactiveV1, TopicDMCheckinV1, TopicDMReplyV1, TopicChatMessage:
		return true
	default:
		return false
	}
}
