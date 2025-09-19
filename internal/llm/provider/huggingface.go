package provider

import ()

type HuggingFaceClient ProviderClient

func newHuggingFaceClient(opts providerClientOptions) OpenAIClient {
	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL("https://api-inference.huggingface.co/v1"),
	)
	return newOpenAIClient(opts)
}

func WithHuggingFaceOptions(huggingfaceOptions ...OpenAIOption) ProviderClientOption {
	return func(options *providerClientOptions) {
		options.openaiOptions = huggingfaceOptions
	}
}
