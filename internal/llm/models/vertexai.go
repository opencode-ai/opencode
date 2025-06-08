package models

const (
	ProviderVertexAI ModelProvider = "vertexai"

	// Models
	VertexAIGemini25FlashPreview520 ModelID = "vertexai.gemini-2.5-flash-preview-05-20"
	VertexAIGemini25FlashPreview417 ModelID = "vertexai.gemini-2.5-flash-preview-04-17"
	VertexAIGemini25ProPreview0605  ModelID = "vertexai.gemini-2.5-pro-preview-06-05"
	VertexAIGemini25ProPreview0506  ModelID = "vertexai.gemini-2.5-pro-preview-05-06"
)

var VertexAIGeminiModels = map[ModelID]Model{
	VertexAIGemini25FlashPreview520: {
		ID:                  VertexAIGemini25FlashPreview520,
		Name:                "VertexAI: Gemini 2.5 Flash Preview (05-20)",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-flash-preview-05-20",
		CostPer1MIn:         GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25Flash].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25FlashPreview417: {
		ID:                  VertexAIGemini25FlashPreview417,
		Name:                "VertexAI: Gemini 2.5 Flash Preview (04-17)",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:         GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25Flash].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25ProPreview0605: {
		ID:                  VertexAIGemini25ProPreview0605,
		Name:                "VertexAI: Gemini 2.5 Pro Preview (06-05)",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-pro-preview-06-05",
		CostPer1MIn:         GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25ProPreview0506: {
		ID:                  VertexAIGemini25ProPreview0506,
		Name:                "VertexAI: Gemini 2.5 Pro Preview (05-06)",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-pro-preview-05-06",
		CostPer1MIn:         GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25].DefaultMaxTokens,
		SupportsAttachments: true,
	},
}
