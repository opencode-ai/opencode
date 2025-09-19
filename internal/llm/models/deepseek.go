package models

const (
	ProviderDeepSeek ModelProvider = "deepseek"

	// DeepSeek Models
	DeepSeekChat     ModelID = "deepseek-chat"
	DeepSeekCoder    ModelID = "deepseek-coder"
	DeepSeekReasoner ModelID = "deepseek-reasoner"
)

// DeepSeek API official models
// https://platform.deepseek.com/api-docs/
// Pricing as of 2025-07-01
var DeepSeekModels = map[ModelID]Model{
	DeepSeekChat: {
		ID:                  DeepSeekChat,
		Name:                "DeepSeek Chat",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-chat",
		CostPer1MIn:         0.14,
		CostPer1MInCached:   0.02,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.28,
		ContextWindow:       128_000, // 官方上限 128k，推荐 ≤ 100k
		DefaultMaxTokens:    8000,    // 官方建议输出 ≤ 8k
		SupportsAttachments: false,   // DeepSeek 目前不支持文件上传或函数调用
	},
	DeepSeekCoder: {
		ID:                  DeepSeekCoder,
		Name:                "DeepSeek Coder",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-coder",
		CostPer1MIn:         0.14,
		CostPer1MInCached:   0.02,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.28,
		ContextWindow:       128_000, // 官方上限 128k，推荐 ≤ 100k
		DefaultMaxTokens:    8000,    // 官方建议输出 ≤ 8k
		SupportsAttachments: false,   // DeepSeek 目前不支持文件上传或函数调用
	},
	DeepSeekReasoner: {
		ID:                  DeepSeekReasoner,
		Name:                "DeepSeek Reasoner (R1)",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-reasoner",
		CostPer1MIn:         0.55,
		CostPer1MInCached:   0.14,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        2.19,
		ContextWindow:       65_536, // R1 模型上下文窗口
		DefaultMaxTokens:    16000,  // R1 建议输出 ≤ 16k
		CanReason:           true,
		SupportsAttachments: false, // DeepSeek 目前不支持文件上传或函数调用
	},
}
