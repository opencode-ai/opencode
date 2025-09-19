package models

const (
	ProviderHuggingFace ModelProvider = "huggingface"
)

const (
	HuggingFaceMistral7BInstruct ModelID = "huggingface-mistral-7b-instruct"
)

var HuggingFaceModels = map[ModelID]Model{
	HuggingFaceMistral7BInstruct: {
		ID:                  HuggingFaceMistral7BInstruct,
		Name:                "Hugging Face: Mistral 7B Instruct",
		Provider:            ProviderHuggingFace,
		APIModel:            "mistralai/Mistral-7B-Instruct-v0.1",
		CostPer1MIn:         0,
		CostPer1MOut:        0,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		ContextWindow:       4096,
		DefaultMaxTokens:    2048,
		CanReason:           true,
		SupportsAttachments: true,
	},
}
