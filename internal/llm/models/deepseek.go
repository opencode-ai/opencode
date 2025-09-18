package models

const (
	ProviderDeepSeek ModelProvider = "deepseek"

	DeepSeekChat     ModelID = "deepseek-chat"
	DeepSeekReasoner ModelID = "deepseek-reasoner"
)

var DeepSeekModels = map[ModelID]Model{
	DeepSeekChat: {
		ID:                  DeepSeekChat,
		Name:                "DeepSeek Chat",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-chat",
		CostPer1MIn:         0.14,
		CostPer1MInCached:   0.014,
		CostPer1MOut:        0.28,
		CostPer1MOutCached:  0.028,
		ContextWindow:       163_840,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	DeepSeekReasoner: {
		ID:                  DeepSeekReasoner,
		Name:                "DeepSeek Reasoner",
		Provider:            ProviderDeepSeek,
		APIModel:            "deepseek-reasoner",
		CostPer1MIn:         0.55,
		CostPer1MInCached:   0.055,
		CostPer1MOut:        2.19,
		CostPer1MOutCached:  0.219,
		ContextWindow:       163_840,
		DefaultMaxTokens:    8192,
		CanReason:           true,
		SupportsAttachments: true, // Reasoner supports tools!
	},
}