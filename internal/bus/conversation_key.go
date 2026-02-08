package bus

import (
	"fmt"
	"strings"
)

type ConversationScope string

const (
	ConversationScopeChat    ConversationScope = "chat"
	ConversationScopeChannel ConversationScope = "channel"
	ConversationScopePeer    ConversationScope = "peer"
	ConversationScopeUser    ConversationScope = "user"
)

func BuildConversationKey(channel Channel, scope ConversationScope, id string) (string, error) {
	if !isValidChannel(channel) {
		return "", fmt.Errorf("channel is invalid")
	}
	if !isValidConversationScope(scope) {
		return "", fmt.Errorf("conversation scope is invalid")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("conversation id is required")
	}
	if strings.Contains(id, " ") {
		return "", fmt.Errorf("conversation id must not contain spaces")
	}
	return fmt.Sprintf("%s:%s:%s", channel, scope, id), nil
}

func BuildTelegramChatConversationKey(chatID string) (string, error) {
	return BuildConversationKey(ChannelTelegram, ConversationScopeChat, chatID)
}

func BuildSlackChannelConversationKey(channelID string) (string, error) {
	return BuildConversationKey(ChannelSlack, ConversationScopeChannel, channelID)
}

func BuildDiscordChannelConversationKey(channelID string) (string, error) {
	return BuildConversationKey(ChannelDiscord, ConversationScopeChannel, channelID)
}

func BuildMAEPPeerConversationKey(peerID string) (string, error) {
	return BuildConversationKey(ChannelMAEP, ConversationScopePeer, peerID)
}

func isValidChannel(channel Channel) bool {
	switch channel {
	case ChannelTelegram, ChannelSlack, ChannelDiscord, ChannelMAEP:
		return true
	default:
		return false
	}
}

func isValidConversationScope(scope ConversationScope) bool {
	switch scope {
	case ConversationScopeChat, ConversationScopeChannel, ConversationScopePeer, ConversationScopeUser:
		return true
	default:
		return false
	}
}
