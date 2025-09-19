package provider

import (
	"os"
)

type OllamaClient ProviderClient

func newOllamaClient(opts providerClientOptions) OpenAIClient {
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL(endpoint),
	)

	return newOpenAIClient(opts)
}

func WithOllamaOptions(ollamaOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = ollamaOptions
	}
}
