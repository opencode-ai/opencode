package provider

import ()

type CohereClient ProviderClient

func newCohereClient(opts providerClientOptions) OpenAIClient {
	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL("https://api.cohere.ai/v1"),
	)
	return newOpenAIClient(opts)
}

func WithCohereOptions(cohereOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = cohereOptions
	}
}
