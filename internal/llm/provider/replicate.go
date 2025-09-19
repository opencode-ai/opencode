package provider

import ()

type ReplicateClient ProviderClient

func newReplicateClient(opts providerClientOptions) OpenAIClient {
	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL("https://api.replicate.com/v1"),
	)
	return newOpenAIClient(opts)
}

func WithReplicateOptions(replicateOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = replicateOptions
	}
}
