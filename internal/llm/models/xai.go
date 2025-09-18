package models

// xAI Model Capabilities (verified via API testing):
// - Reasoning support:
//   - grok-4-0709: Has internal reasoning capabilities but does NOT expose reasoning_content or accept reasoning_effort parameter
//   - grok-3-mini, grok-3-mini-fast: Support reasoning_effort parameter ("low" or "high" only, NOT "medium")
//   - grok-2 models, grok-3, grok-3-fast: No reasoning support
// - Vision support: grok-2-vision-1212 and grok-4 support image understanding
// - Image generation: grok-2-image and potentially grok-4 support image generation
// - Web search: All models support web search via tools
// - Note: Reasoning models cannot use presencePenalty, frequencyPenalty, or stop parameters

const (
	ProviderXAI ModelProvider = "xai"

	// Current xAI models (from API as of 2025)
	XAIGrok2         ModelID = "grok-2-1212"
	XAIGrok2Vision   ModelID = "grok-2-vision-1212"
	XAIGrok2Image    ModelID = "grok-2-image-1212"
	XAIGrok3         ModelID = "grok-3"
	XAIGrok3Fast     ModelID = "grok-3-fast"
	XAIGrok3Mini     ModelID = "grok-3-mini"
	XAIGrok3MiniFast ModelID = "grok-3-mini-fast"
	XAIGrok4         ModelID = "grok-4-0709"
)

var XAIModels = map[ModelID]Model{
	XAIGrok2: {
		ID:                 XAIGrok2,
		Name:               "Grok 2",
		Provider:           ProviderXAI,
		APIModel:           "grok-2-1212",
		CostPer1MIn:        2.0, // $2 per million input tokens
		CostPer1MInCached:  0,
		CostPer1MOut:       10.0, // $10 per million output tokens
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
		CanReason:          false, // No reasoning support
		// Capabilities: streaming, function calling, structured outputs, web search
	},
	XAIGrok2Vision: {
		ID:                  XAIGrok2Vision,
		Name:                "Grok 2 Vision",
		Provider:            ProviderXAI,
		APIModel:            "grok-2-vision-1212",
		CostPer1MIn:         2.0, // $2 per million input tokens
		CostPer1MInCached:   0,
		CostPer1MOut:        10.0, // $10 per million output tokens
		CostPer1MOutCached:  0,
		ContextWindow:       8_192,
		DefaultMaxTokens:    4_096,
		SupportsAttachments: true,
		CanReason:           false, // No reasoning support
		// Capabilities: image understanding, streaming, web search
	},
	XAIGrok2Image: {
		ID:                      XAIGrok2Image,
		Name:                    "Grok 2 Image",
		Provider:                ProviderXAI,
		APIModel:                "grok-2-image-1212",
		CostPer1MIn:             2.0, // Assuming same as Grok 2
		CostPer1MInCached:       0,
		CostPer1MOut:            10.0, // Assuming same as Grok 2
		CostPer1MOutCached:      0,
		ContextWindow:           8_192,
		DefaultMaxTokens:        4_096,
		SupportsAttachments:     false, // Image generation models don't take image inputs
		SupportsImageGeneration: true,
		// Capabilities: image generation, web search
	},
	XAIGrok3: {
		ID:                 XAIGrok3,
		Name:               "Grok 3",
		Provider:           ProviderXAI,
		APIModel:           "grok-3",
		CostPer1MIn:        5.0, // Estimated pricing
		CostPer1MInCached:  0,
		CostPer1MOut:       15.0, // Estimated pricing
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
		CanReason:          false, // No reasoning support
		// Capabilities: streaming, function calling, structured outputs, web search
	},
	XAIGrok3Fast: {
		ID:                 XAIGrok3Fast,
		Name:               "Grok 3 Fast",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-fast",
		CostPer1MIn:        3.0, // Estimated lower pricing for fast variant
		CostPer1MInCached:  0,
		CostPer1MOut:       10.0, // Estimated lower pricing for fast variant
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
		CanReason:          false, // No reasoning support
		// Capabilities: streaming, function calling, structured outputs, web search
	},
	XAIGrok3Mini: {
		ID:                 XAIGrok3Mini,
		Name:               "Grok 3 Mini",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini",
		CostPer1MIn:        1.0, // Estimated lower pricing for mini
		CostPer1MInCached:  0,
		CostPer1MOut:       3.0, // Estimated lower pricing for mini
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
		CanReason:          true, // Supports reasoning_effort parameter ("low" or "high")
		// Capabilities: streaming, function calling, structured outputs, reasoning, web search
	},
	XAIGrok3MiniFast: {
		ID:                 XAIGrok3MiniFast,
		Name:               "Grok 3 Mini Fast",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini-fast",
		CostPer1MIn:        0.5, // Estimated lowest pricing
		CostPer1MInCached:  0,
		CostPer1MOut:       1.5, // Estimated lowest pricing
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
		CanReason:          true, // Supports reasoning_effort parameter ("low" or "high")
		// Capabilities: streaming, function calling, structured outputs, reasoning, web search
	},
	XAIGrok4: {
		ID:                      XAIGrok4,
		Name:                    "Grok 4",
		Provider:                ProviderXAI,
		APIModel:                "grok-4-0709",
		CostPer1MIn:             10.0, // $10 per million input tokens
		CostPer1MInCached:       0,
		CostPer1MOut:            30.0, // $30 per million output tokens
		CostPer1MOutCached:      0,
		ContextWindow:           131_072,
		DefaultMaxTokens:        20_000,
		CanReason:               true, // Has reasoning capabilities but doesn't expose reasoning content
		SupportsAttachments:     true,  // Grok 4 supports vision
		SupportsImageGeneration: false, // Will be detected dynamically via API
		// Capabilities: streaming, function calling, structured outputs, web search, vision
	},
}
