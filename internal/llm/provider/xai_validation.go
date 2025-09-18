package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opencode-ai/opencode/internal/logging"
)

// XAIAPIKeyInfo represents the information about an xAI API key
type XAIAPIKeyInfo struct {
	RedactedAPIKey string   `json:"redacted_api_key"`
	UserID         string   `json:"user_id"`
	Name           string   `json:"name"`
	CreateTime     string   `json:"create_time"`
	ModifyTime     string   `json:"modify_time"`
	ModifiedBy     string   `json:"modified_by"`
	TeamID         string   `json:"team_id"`
	ACLs           []string `json:"acls"`
	APIKeyID       string   `json:"api_key_id"`
	TeamBlocked    bool     `json:"team_blocked"`
	APIKeyBlocked  bool     `json:"api_key_blocked"`
	APIKeyDisabled bool     `json:"api_key_disabled"`
}

// ValidateAPIKey validates the xAI API key and returns detailed information about it
func (x *xaiClient) ValidateAPIKey(ctx context.Context) (*XAIAPIKeyInfo, error) {
	url := fmt.Sprintf("%s/api-key", x.getBaseURL())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+x.providerOptions.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read validation response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API key validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var keyInfo XAIAPIKeyInfo
	if err := json.Unmarshal(body, &keyInfo); err != nil {
		return nil, fmt.Errorf("failed to parse API key info: %w", err)
	}

	logging.Debug("xAI API key validation successful",
		"redacted_key", keyInfo.RedactedAPIKey,
		"name", keyInfo.Name,
		"team_id", keyInfo.TeamID,
		"blocked", keyInfo.APIKeyBlocked,
		"disabled", keyInfo.APIKeyDisabled,
		"team_blocked", keyInfo.TeamBlocked)

	return &keyInfo, nil
}

// IsAPIKeyValid performs a quick validation check and returns true if the key is valid and active
func (x *xaiClient) IsAPIKeyValid(ctx context.Context) bool {
	keyInfo, err := x.ValidateAPIKey(ctx)
	if err != nil {
		logging.Debug("API key validation failed", "error", err)
		return false
	}

	// Check if key is blocked or disabled
	if keyInfo.APIKeyBlocked || keyInfo.APIKeyDisabled || keyInfo.TeamBlocked {
		logging.Warn("xAI API key is blocked or disabled",
			"api_key_blocked", keyInfo.APIKeyBlocked,
			"api_key_disabled", keyInfo.APIKeyDisabled,
			"team_blocked", keyInfo.TeamBlocked)
		return false
	}

	return true
}

// CheckPermissions validates that the API key has the required permissions for specific operations
func (x *xaiClient) CheckPermissions(ctx context.Context, requiredACLs []string) error {
	keyInfo, err := x.ValidateAPIKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check if key is blocked or disabled
	if keyInfo.APIKeyBlocked {
		return fmt.Errorf("API key is blocked")
	}
	if keyInfo.APIKeyDisabled {
		return fmt.Errorf("API key is disabled")
	}
	if keyInfo.TeamBlocked {
		return fmt.Errorf("team is blocked")
	}

	// Check if required ACLs are present
	aclMap := make(map[string]bool)
	for _, acl := range keyInfo.ACLs {
		aclMap[acl] = true
	}

	var missingACLs []string
	for _, required := range requiredACLs {
		if !aclMap[required] && !aclMap["api-key:endpoint:*"] && !aclMap["api-key:model:*"] {
			// Check for wildcard permissions
			found := false
			for _, acl := range keyInfo.ACLs {
				if acl == "api-key:endpoint:*" || acl == "api-key:model:*" {
					found = true
					break
				}
			}
			if !found {
				missingACLs = append(missingACLs, required)
			}
		}
	}

	if len(missingACLs) > 0 {
		return fmt.Errorf("API key missing required permissions: %v", missingACLs)
	}

	logging.Debug("xAI API key permissions validated",
		"required_acls", requiredACLs,
		"available_acls", keyInfo.ACLs)

	return nil
}

// GetAPIKeyInfo returns detailed information about the API key for debugging purposes
func (x *xaiClient) GetAPIKeyInfo(ctx context.Context) (map[string]interface{}, error) {
	keyInfo, err := x.ValidateAPIKey(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"redacted_key": keyInfo.RedactedAPIKey,
		"name":         keyInfo.Name,
		"team_id":      keyInfo.TeamID,
		"created":      keyInfo.CreateTime,
		"modified":     keyInfo.ModifyTime,
		"permissions":  keyInfo.ACLs,
		"status": map[string]interface{}{
			"active":       !keyInfo.APIKeyBlocked && !keyInfo.APIKeyDisabled && !keyInfo.TeamBlocked,
			"key_blocked":  keyInfo.APIKeyBlocked,
			"key_disabled": keyInfo.APIKeyDisabled,
			"team_blocked": keyInfo.TeamBlocked,
		},
	}, nil
}

// ValidateForOperation checks if the API key is valid for a specific operation type
func (x *xaiClient) ValidateForOperation(ctx context.Context, operation string) error {
	var requiredACLs []string

	switch operation {
	case "chat":
		requiredACLs = []string{"api-key:endpoint:chat", "api-key:model:*"}
	case "image_generation":
		requiredACLs = []string{"api-key:endpoint:images", "api-key:model:*"}
	case "models":
		requiredACLs = []string{"api-key:endpoint:models"}
	default:
		// For unknown operations, just check basic endpoint access
		requiredACLs = []string{"api-key:endpoint:*"}
	}

	return x.CheckPermissions(ctx, requiredACLs)
}

// HealthCheck performs a comprehensive health check of the xAI API key and service
func (x *xaiClient) HealthCheck(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"provider":  "xai",
		"model":     string(x.providerOptions.model.ID),
	}

	// Test API key validation
	keyInfo, err := x.ValidateAPIKey(ctx)
	if err != nil {
		result["api_key_status"] = "invalid"
		result["api_key_error"] = err.Error()
		result["overall_status"] = "failed"
		return result
	}

	result["api_key_status"] = "valid"
	result["api_key_name"] = keyInfo.Name
	result["team_id"] = keyInfo.TeamID

	// Check if key is active
	if keyInfo.APIKeyBlocked || keyInfo.APIKeyDisabled || keyInfo.TeamBlocked {
		result["key_active"] = false
		result["block_reasons"] = map[string]bool{
			"api_key_blocked":  keyInfo.APIKeyBlocked,
			"api_key_disabled": keyInfo.APIKeyDisabled,
			"team_blocked":     keyInfo.TeamBlocked,
		}
		result["overall_status"] = "blocked"
		return result
	}

	result["key_active"] = true
	result["permissions"] = keyInfo.ACLs

	// Test model capabilities if available
	caps, err := x.DiscoverModelCapabilities(ctx, string(x.providerOptions.model.ID))
	if err != nil {
		result["model_capabilities"] = "unavailable"
		result["capabilities_error"] = err.Error()
	} else {
		result["model_capabilities"] = map[string]interface{}{
			"supports_text":         caps.SupportsText,
			"supports_image_input":  caps.SupportsImageInput,
			"supports_image_output": caps.SupportsImageOutput,
			"supports_web_search":   caps.SupportsWebSearch,
			"max_prompt_length":     caps.MaxPromptLength,
			"aliases":               caps.Aliases,
		}
	}

	result["overall_status"] = "healthy"
	return result
}
