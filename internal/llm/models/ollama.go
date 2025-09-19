package models

const (
	ProviderOllama ModelProvider = "ollama"
)

const (
	OllamaLlama3 ModelID = "ollama-llama3"
)

var OllamaModels = map[ModelID]Model{
	OllamaLlama3: {
		ID:                  OllamaLlama3,
		Name:                "Ollama: Llama 3",
		Provider:            ProviderOllama,
		APIModel:            "llama3",
		CostPer1MIn:         0,
		CostPer1MOut:        0,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		ContextWindow:       8192,
		DefaultMaxTokens:    4096,
		CanReason:           true,
		SupportsAttachments: true,
	},
}
