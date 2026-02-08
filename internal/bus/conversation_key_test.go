package bus

import "testing"

func TestBuildConversationKey(t *testing.T) {
	key, err := BuildConversationKey(ChannelTelegram, ConversationScopeChat, "-1001")
	if err != nil {
		t.Fatalf("BuildConversationKey() error = %v", err)
	}
	if key != "telegram:chat:-1001" {
		t.Fatalf("conversation key mismatch: got %q", key)
	}
}

func TestBuildConversationKeyRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		channel Channel
		scope   ConversationScope
		id      string
	}{
		{name: "invalid channel", channel: Channel("unknown"), scope: ConversationScopeChat, id: "1"},
		{name: "invalid scope", channel: ChannelTelegram, scope: ConversationScope("xxx"), id: "1"},
		{name: "empty id", channel: ChannelTelegram, scope: ConversationScopeChat, id: "   "},
		{name: "id contains space", channel: ChannelTelegram, scope: ConversationScopeChat, id: "a b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildConversationKey(tc.channel, tc.scope, tc.id); err == nil {
				t.Fatalf("BuildConversationKey() expected error")
			}
		})
	}
}
