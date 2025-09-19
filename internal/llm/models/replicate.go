package models

const (
	ProviderReplicate ModelProvider = "replicate"
)

const (
	ReplicateLlama270BChat ModelID = "replicate-llama-2-70b-chat"
)

var ReplicateModels = map[ModelID]Model{
	ReplicateLlama270BChat: {
		ID:                  ReplicateLlama270BChat,
		Name:                "Replicate: Llama 2 70B Chat",
		Provider:            ProviderReplicate,
		APIModel:            "meta/llama-2-70b-chat:2796ee9483c3fd7aa2e171d38f4ca12251a30609463dcfd4cd76703f22e96e2e",
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
