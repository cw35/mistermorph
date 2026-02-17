package llmutil

import (
	"strings"
	"testing"
	"time"

	"github.com/quailyquaily/mistermorph/internal/llmconfig"
)

func TestEndpointForProviderWithValues_CloudflareDefaultEndpoint(t *testing.T) {
	values := RuntimeValues{Endpoint: "https://api.openai.com"}
	if got := EndpointForProviderWithValues("cloudflare", values); got != "" {
		t.Fatalf("EndpointForProviderWithValues() = %q, want empty", got)
	}
	values.Endpoint = "https://api.openai.com/v1"
	if got := EndpointForProviderWithValues("cloudflare", values); got != "" {
		t.Fatalf("EndpointForProviderWithValues() = %q, want empty", got)
	}
}

func TestEndpointForProviderWithValues_CloudflareCustomEndpoint(t *testing.T) {
	values := RuntimeValues{Endpoint: "https://gateway.ai.cloudflare.com/v1/acc/route"}
	if got := EndpointForProviderWithValues("cloudflare", values); got != values.Endpoint {
		t.Fatalf("EndpointForProviderWithValues() = %q, want %q", got, values.Endpoint)
	}
}

func TestAPIKeyForProviderWithValues_CloudflareFallback(t *testing.T) {
	values := RuntimeValues{
		APIKey:             "generic-key",
		CloudflareAPIToken: "cf-token",
	}
	if got := APIKeyForProviderWithValues("cloudflare", values); got != "cf-token" {
		t.Fatalf("APIKeyForProviderWithValues() = %q, want cf-token", got)
	}
	values.CloudflareAPIToken = ""
	if got := APIKeyForProviderWithValues("cloudflare", values); got != "generic-key" {
		t.Fatalf("APIKeyForProviderWithValues() = %q, want generic-key", got)
	}
}

func TestModelForProviderWithValues_AzureDeploymentFirst(t *testing.T) {
	values := RuntimeValues{
		Model:           "gpt-5.2",
		AzureDeployment: "gpt5-deploy",
	}
	if got := ModelForProviderWithValues("azure", values); got != "gpt5-deploy" {
		t.Fatalf("ModelForProviderWithValues() = %q, want gpt5-deploy", got)
	}
	values.AzureDeployment = ""
	if got := ModelForProviderWithValues("azure", values); got != "gpt-5.2" {
		t.Fatalf("ModelForProviderWithValues() = %q, want gpt-5.2", got)
	}
}

func TestClientFromConfigWithValues_InvalidToolsMode(t *testing.T) {
	_, err := ClientFromConfigWithValues(llmconfig.ClientConfig{
		Provider:       "openai",
		Endpoint:       "https://api.openai.com",
		APIKey:         "k",
		Model:          "gpt-5.2",
		RequestTimeout: 10 * time.Second,
	}, RuntimeValues{
		ToolsEmulationMode: "invalid",
	})
	if err == nil {
		t.Fatalf("expected error for invalid tools emulation mode")
	}
	if !strings.Contains(err.Error(), "llm.tools_emulation_mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}
