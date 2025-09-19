package models

const (
	ProviderCohere ModelProvider = "cohere"
)

const (
	CohereCommandRPlus ModelID = "cohere-command-r-plus"
)

var CohereModels = map[ModelID]Model{
	CohereCommandRPlus: {
		ID:                  CohereCommandRPlus,
		Name:                "Cohere: Command R+",
		Provider:            ProviderCohere,
		APIModel:            "command-r-plus",
		CostPer1MIn:         0,
		CostPer1MOut:        0,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		CanReason:           true,
		SupportsAttachments: true,
	},
}
