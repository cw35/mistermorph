package bus

import "testing"

func TestStartInproc(t *testing.T) {
	b, err := StartInproc(BootstrapOptions{
		MaxInFlight: 4,
		Logger:      newTestLogger(),
		Component:   "test",
	})
	if err != nil {
		t.Fatalf("StartInproc() error = %v", err)
	}
	if b == nil {
		t.Fatalf("StartInproc() bus should not be nil")
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestStartInprocRequiresLogger(t *testing.T) {
	if _, err := StartInproc(BootstrapOptions{MaxInFlight: 4}); err == nil {
		t.Fatalf("StartInproc() expected error when logger is nil")
	}
}
