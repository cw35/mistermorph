package llmutil

import (
	"fmt"
	"strings"

	"github.com/quailyquaily/mistermorph/internal/llmconfig"
	"github.com/quailyquaily/mistermorph/llm"
	uniaiProvider "github.com/quailyquaily/mistermorph/providers/uniai"
	"github.com/spf13/viper"
)

type ConfigReader interface {
	GetString(string) string
}

type RuntimeValues struct {
	Provider           string
	Endpoint           string
	APIKey             string
	Model              string
	AzureDeployment    string
	ToolsEmulationMode string

	BedrockAWSKey       string
	BedrockAWSSecret    string
	BedrockAWSRegion    string
	BedrockModelARN     string
	CloudflareAccountID string
	CloudflareAPIToken  string
}

func RuntimeValuesFromReader(r ConfigReader) RuntimeValues {
	if r == nil {
		return RuntimeValues{}
	}
	return RuntimeValues{
		Provider:            strings.TrimSpace(r.GetString("llm.provider")),
		Endpoint:            strings.TrimSpace(r.GetString("llm.endpoint")),
		APIKey:              strings.TrimSpace(r.GetString("llm.api_key")),
		Model:               strings.TrimSpace(r.GetString("llm.model")),
		AzureDeployment:     strings.TrimSpace(r.GetString("llm.azure.deployment")),
		ToolsEmulationMode:  strings.TrimSpace(r.GetString("llm.tools_emulation_mode")),
		BedrockAWSKey:       firstNonEmpty(r.GetString("llm.bedrock.aws_key"), r.GetString("llm.aws.key")),
		BedrockAWSSecret:    firstNonEmpty(r.GetString("llm.bedrock.aws_secret"), r.GetString("llm.aws.secret")),
		BedrockAWSRegion:    firstNonEmpty(r.GetString("llm.bedrock.region"), r.GetString("llm.aws.region")),
		BedrockModelARN:     firstNonEmpty(r.GetString("llm.bedrock.model_arn"), r.GetString("llm.aws.bedrock_model_arn")),
		CloudflareAccountID: firstNonEmpty(r.GetString("llm.cloudflare.account_id")),
		CloudflareAPIToken:  firstNonEmpty(r.GetString("llm.cloudflare.api_token")),
	}
}

func RuntimeValuesFromViper() RuntimeValues {
	return RuntimeValuesFromReader(viper.GetViper())
}

func ProviderFromViper() string {
	return strings.TrimSpace(RuntimeValuesFromViper().Provider)
}

func EndpointFromViper() string {
	values := RuntimeValuesFromViper()
	return EndpointForProviderWithValues(values.Provider, values)
}

func APIKeyFromViper() string {
	values := RuntimeValuesFromViper()
	return APIKeyForProviderWithValues(values.Provider, values)
}

func ModelFromViper() string {
	values := RuntimeValuesFromViper()
	return ModelForProviderWithValues(values.Provider, values)
}

func EndpointForProvider(provider string) string {
	return EndpointForProviderWithValues(provider, RuntimeValuesFromViper())
}

func EndpointForProviderWithValues(provider string, values RuntimeValues) string {
	provider = normalizeProvider(provider)
	switch provider {
	case "cloudflare":
		generic := strings.TrimSpace(values.Endpoint)
		if generic != "" && generic != "https://api.openai.com" && generic != "https://api.openai.com/v1" {
			return generic
		}
		return ""
	default:
		return strings.TrimSpace(values.Endpoint)
	}
}

func APIKeyForProvider(provider string) string {
	return APIKeyForProviderWithValues(provider, RuntimeValuesFromViper())
}

func APIKeyForProviderWithValues(provider string, values RuntimeValues) string {
	provider = normalizeProvider(provider)
	switch provider {
	case "cloudflare":
		return firstNonEmpty(values.CloudflareAPIToken, values.APIKey)
	default:
		return strings.TrimSpace(values.APIKey)
	}
}

func ModelForProvider(provider string) string {
	return ModelForProviderWithValues(provider, RuntimeValuesFromViper())
}

func ModelForProviderWithValues(provider string, values RuntimeValues) string {
	provider = normalizeProvider(provider)
	switch provider {
	case "azure":
		return firstNonEmpty(
			values.AzureDeployment,
			values.Model,
		)
	default:
		return strings.TrimSpace(values.Model)
	}
}

func ClientFromConfig(cfg llmconfig.ClientConfig) (llm.Client, error) {
	return ClientFromConfigWithValues(cfg, RuntimeValuesFromViper())
}

func ClientFromConfigWithValues(cfg llmconfig.ClientConfig, values RuntimeValues) (llm.Client, error) {
	toolsEmulationMode, err := toolsEmulationModeFromValue(values.ToolsEmulationMode)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "openai", "openai_custom", "deepseek", "xai", "gemini", "azure", "anthropic", "bedrock", "susanoo", "cloudflare":
		c := uniaiProvider.New(uniaiProvider.Config{
			Provider:           strings.ToLower(strings.TrimSpace(cfg.Provider)),
			Endpoint:           strings.TrimSpace(cfg.Endpoint),
			APIKey:             strings.TrimSpace(cfg.APIKey),
			Model:              strings.TrimSpace(cfg.Model),
			RequestTimeout:     cfg.RequestTimeout,
			ToolsEmulationMode: toolsEmulationMode,
			AzureAPIKey:        strings.TrimSpace(cfg.APIKey),
			AzureEndpoint:      strings.TrimSpace(cfg.Endpoint),
			AzureDeployment:    strings.TrimSpace(cfg.Model),
			AwsKey:             firstNonEmpty(values.BedrockAWSKey),
			AwsSecret:          firstNonEmpty(values.BedrockAWSSecret),
			AwsRegion:          firstNonEmpty(values.BedrockAWSRegion),
			AwsBedrockModelArn: firstNonEmpty(values.BedrockModelARN),
			CloudflareAccountID: firstNonEmpty(
				values.CloudflareAccountID,
			),
			CloudflareAPIToken: firstNonEmpty(
				values.CloudflareAPIToken,
				values.APIKey,
			),
			CloudflareAPIBase: strings.TrimSpace(cfg.Endpoint),
		})
		return c, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

func toolsEmulationModeFromViper() (string, error) {
	return toolsEmulationModeFromValue(RuntimeValuesFromViper().ToolsEmulationMode)
}

func toolsEmulationModeFromValue(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return "off", nil
	}
	switch mode {
	case "off", "fallback", "force":
		return mode, nil
	default:
		return "", fmt.Errorf("invalid llm.tools_emulation_mode %q (expected off|fallback|force)", mode)
	}
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return "openai"
	}
	return provider
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
