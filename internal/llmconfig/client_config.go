package llmconfig

import "time"

type ClientConfig struct {
	Provider       string
	Endpoint       string
	APIKey         string
	Model          string
	RequestTimeout time.Duration
}
