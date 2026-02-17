package grouptrigger

import (
	"context"
	"errors"
	"testing"
)

func TestDecideExplicitMatched(t *testing.T) {
	t.Parallel()

	called := false
	dec, ok, err := Decide(context.Background(), DecideOptions{
		Mode:            "smart",
		ExplicitReason:  "mention",
		ExplicitMatched: true,
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			called = true
			return Addressing{}, false, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if !ok {
		t.Fatalf("Decide() ok=false, want true")
	}
	if called {
		t.Fatalf("addressing should not be called on explicit match")
	}
	if dec.Reason != "mention" {
		t.Fatalf("reason mismatch: got %q want %q", dec.Reason, "mention")
	}
	if dec.Addressing.Impulse != 1 {
		t.Fatalf("impulse mismatch: got %v want 1", dec.Addressing.Impulse)
	}
}

func TestDecideStrictWithoutExplicit(t *testing.T) {
	t.Parallel()

	called := false
	_, ok, err := Decide(context.Background(), DecideOptions{
		Mode: "strict",
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			called = true
			return Addressing{}, false, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if ok {
		t.Fatalf("Decide() ok=true, want false")
	}
	if called {
		t.Fatalf("addressing should not be called in strict mode without explicit trigger")
	}
}

func TestDecideSmartRequiresAddressed(t *testing.T) {
	t.Parallel()

	_, ok, err := Decide(context.Background(), DecideOptions{
		Mode:                "smart",
		ConfidenceThreshold: 0.6,
		InterjectThreshold:  0.5,
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			return Addressing{
				Addressed:  false,
				Confidence: 0.9,
				Interject:  0.9,
				Impulse:    0.9,
			}, true, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if ok {
		t.Fatalf("Decide() ok=true, want false")
	}
}

func TestDecideSmartIgnoresInterject(t *testing.T) {
	t.Parallel()

	_, ok, err := Decide(context.Background(), DecideOptions{
		Mode:                "smart",
		ConfidenceThreshold: 0.6,
		InterjectThreshold:  0.9,
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			return Addressing{
				Addressed:  true,
				Confidence: 0.95,
				Interject:  0.1,
			}, true, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if !ok {
		t.Fatalf("Decide() ok=false, want true")
	}
}

func TestDecideTalkativeUsesWannaInterjectAndInterject(t *testing.T) {
	t.Parallel()

	_, ok, err := Decide(context.Background(), DecideOptions{
		Mode:                "talkative",
		ConfidenceThreshold: 0.95,
		InterjectThreshold:  0.6,
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			return Addressing{
				Addressed:      false,
				Confidence:     0.1,
				WannaInterject: true,
				Interject:      0.8,
			}, true, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if !ok {
		t.Fatalf("Decide() ok=false, want true")
	}
}

func TestDecideTalkativeRejectsWithoutWannaInterject(t *testing.T) {
	t.Parallel()

	_, ok, err := Decide(context.Background(), DecideOptions{
		Mode:               "talkative",
		InterjectThreshold: 0.6,
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			return Addressing{
				WannaInterject: false,
				Interject:      0.9,
			}, true, nil
		},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if ok {
		t.Fatalf("Decide() ok=true, want false")
	}
}

func TestDecideAddressingError(t *testing.T) {
	t.Parallel()

	expected := errors.New("boom")
	_, _, err := Decide(context.Background(), DecideOptions{
		Mode: "smart",
		Addressing: func(ctx context.Context) (Addressing, bool, error) {
			return Addressing{}, false, expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatalf("Decide() error = %v, want %v", err, expected)
	}
}
