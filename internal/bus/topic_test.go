package bus

import "testing"

func TestValidateTopic(t *testing.T) {
	if err := ValidateTopic(TopicChatMessage); err != nil {
		t.Fatalf("ValidateTopic() error = %v", err)
	}
	if err := ValidateTopic("agent.status.v1"); err == nil {
		t.Fatalf("ValidateTopic() expected error for unknown topic")
	}
}
