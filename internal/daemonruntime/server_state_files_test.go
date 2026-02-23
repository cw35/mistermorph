package daemonruntime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestRuntimeStateFileSpecsIncludesHeartbeat(t *testing.T) {
	paths := runtimeStatePaths{
		todoWIP:          "/tmp/TODO.md",
		todoDone:         "/tmp/TODO.DONE.md",
		contactsActive:   "/tmp/ACTIVE.md",
		contactsInactive: "/tmp/INACTIVE.md",
		identityPath:     "/tmp/IDENTITY.md",
		soulPath:         "/tmp/SOUL.md",
		heartbeatPath:    "/tmp/HEARTBEAT.md",
	}

	items := describeStateFiles(paths, "")
	if len(items) != 7 {
		t.Fatalf("len(items) = %d, want 7", len(items))
	}

	foundHeartbeat := false
	for _, item := range items {
		if item["name"] == "HEARTBEAT.md" && item["group"] == "heartbeat" {
			foundHeartbeat = true
			break
		}
	}
	if !foundHeartbeat {
		t.Fatalf("HEARTBEAT.md should be present in state files: %#v", items)
	}
}

func TestResolveStateFileSpec(t *testing.T) {
	paths := runtimeStatePaths{
		todoWIP:          "/tmp/TODO.md",
		todoDone:         "/tmp/TODO.DONE.md",
		contactsActive:   "/tmp/ACTIVE.md",
		contactsInactive: "/tmp/INACTIVE.md",
		identityPath:     "/tmp/IDENTITY.md",
		soulPath:         "/tmp/SOUL.md",
		heartbeatPath:    "/tmp/HEARTBEAT.md",
	}

	if spec, ok := resolveStateFileSpec(paths, "", "heartbeat.md"); !ok || spec.Group != "heartbeat" {
		t.Fatalf("resolve heartbeat failed: ok=%v spec=%#v", ok, spec)
	}
	if _, ok := resolveStateFileSpec(paths, "todo", "ACTIVE.md"); ok {
		t.Fatalf("resolve with wrong group should fail")
	}
	if spec, ok := resolveStateFileSpec(paths, "todo", "todo.md"); !ok || spec.Name != "TODO.md" {
		t.Fatalf("resolve todo failed: ok=%v spec=%#v", ok, spec)
	}
}

func TestStateFilesRoute(t *testing.T) {
	stateDir := t.TempDir()
	oldStateDir := viper.GetString("file_state_dir")
	oldContactsDir := viper.GetString("contacts.dir_name")
	t.Cleanup(func() {
		viper.Set("file_state_dir", oldStateDir)
		viper.Set("contacts.dir_name", oldContactsDir)
	})
	viper.Set("file_state_dir", stateDir)
	viper.Set("contacts.dir_name", "contacts")

	mux := http.NewServeMux()
	RegisterRoutes(mux, RoutesOptions{
		Mode:      "serve",
		AuthToken: "token",
	})

	req := httptest.NewRequest(http.MethodGet, "/state/files", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(payload.Items) != 7 {
		t.Fatalf("len(items) = %d, want 7", len(payload.Items))
	}
}
