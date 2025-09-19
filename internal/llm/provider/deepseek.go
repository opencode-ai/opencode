package provider

type deepseekClient struct {
	*openaiClient
}

type DeepSeekClient ProviderClient

func newDeepSeekClient(opts providerClientOptions) DeepSeekClient {
	// DeepSeek API 的基础 URL
	baseURL := "https://api.deepseek.com"

	// 将基础 URL 添加到 OpenAI 客户端选项中
	opts.openaiOptions = append(opts.openaiOptions,
		WithOpenAIBaseURL(baseURL),
	)

	// 创建并返回一个包装了 openaiClient 的 deepseekClient
	return &deepseekClient{
		openaiClient: newOpenAIClient(opts).(*openaiClient),
	}
}

// DeepSeek 客户端实际上就是 OpenAI 客户端，只是指向不同的 API 端点
// 所有方法都通过嵌入的 openaiClient 来处理
