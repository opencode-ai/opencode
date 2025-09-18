package provider

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/llm/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIProvider_APIKeyValidation(t *testing.T) {
	// Skip if no API key is provided
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}

	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("validate API key", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Validate API key
		keyInfo, err := xaiClient.ValidateAPIKey(ctx)
		require.NoError(t, err)
		require.NotNil(t, keyInfo)

		// Check basic fields
		assert.NotEmpty(t, keyInfo.RedactedAPIKey, "Should have redacted API key")
		assert.NotEmpty(t, keyInfo.UserID, "Should have user ID")
		assert.NotEmpty(t, keyInfo.TeamID, "Should have team ID")
		assert.NotEmpty(t, keyInfo.APIKeyID, "Should have API key ID")
		assert.NotEmpty(t, keyInfo.ACLs, "Should have ACLs")

		// Check key status
		assert.False(t, keyInfo.APIKeyBlocked, "API key should not be blocked")
		assert.False(t, keyInfo.APIKeyDisabled, "API key should not be disabled")
		assert.False(t, keyInfo.TeamBlocked, "Team should not be blocked")

		t.Logf("API Key Info:")
		t.Logf("  Redacted Key: %s", keyInfo.RedactedAPIKey)
		t.Logf("  Name: %s", keyInfo.Name)
		t.Logf("  Team ID: %s", keyInfo.TeamID)
		t.Logf("  Created: %s", keyInfo.CreateTime)
		t.Logf("  ACLs: %v", keyInfo.ACLs)
	})

	t.Run("check if API key is valid", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Check if key is valid
		isValid := xaiClient.IsAPIKeyValid(ctx)
		assert.True(t, isValid, "API key should be valid")

		t.Logf("API key validation result: %v", isValid)
	})

	t.Run("check permissions", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Check basic permissions
		err = xaiClient.CheckPermissions(ctx, []string{"api-key:model:*"})
		assert.NoError(t, err, "Should have model permissions")

		err = xaiClient.CheckPermissions(ctx, []string{"api-key:endpoint:*"})
		assert.NoError(t, err, "Should have endpoint permissions")

		t.Log("Permission checks passed")
	})

	t.Run("validate for specific operations", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Test different operations
		operations := []string{"chat", "image_generation", "models"}
		for _, op := range operations {
			err = xaiClient.ValidateForOperation(ctx, op)
			if err != nil {
				t.Logf("Operation %s validation failed: %v", op, err)
			} else {
				t.Logf("Operation %s validation passed", op)
			}
		}
	})

	t.Run("get API key info", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Get API key info
		info, err := xaiClient.GetAPIKeyInfo(ctx)
		require.NoError(t, err)
		require.NotNil(t, info)

		// Check required fields
		assert.Contains(t, info, "redacted_key")
		assert.Contains(t, info, "name")
		assert.Contains(t, info, "team_id")
		assert.Contains(t, info, "permissions")
		assert.Contains(t, info, "status")

		status, ok := info["status"].(map[string]interface{})
		require.True(t, ok, "Status should be a map")
		assert.Contains(t, status, "active")

		t.Logf("API key info retrieved successfully")
		t.Logf("  Active: %v", status["active"])
		t.Logf("  Permissions: %v", info["permissions"])
	})

	t.Run("health check", func(t *testing.T) {
		// Create provider
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(apiKey),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Perform health check
		health := xaiClient.HealthCheck(ctx)
		require.NotNil(t, health)

		// Check required fields
		assert.Contains(t, health, "timestamp")
		assert.Contains(t, health, "provider")
		assert.Contains(t, health, "model")
		assert.Contains(t, health, "overall_status")
		assert.Contains(t, health, "api_key_status")

		assert.Equal(t, "xai", health["provider"])
		assert.Equal(t, "grok-3-fast", health["model"])

		overallStatus := health["overall_status"]
		t.Logf("Health check status: %v", overallStatus)

		// Should be healthy or at least not failed
		assert.NotEqual(t, "failed", overallStatus, "Health check should not fail completely")

		if overallStatus == "healthy" {
			assert.Equal(t, "valid", health["api_key_status"])
			assert.True(t, health["key_active"].(bool))
		}

		t.Logf("Health check completed: %v", health)
	})
}

func TestXAIProvider_InvalidAPIKey(t *testing.T) {
	// Initialize config for tests
	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	require.NoError(t, err)

	t.Run("invalid API key", func(t *testing.T) {
		// Create provider with invalid API key
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey("xai-invalid-key-12345"),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Validate API key should fail
		_, err = xaiClient.ValidateAPIKey(ctx)
		assert.Error(t, err, "Should fail with invalid API key")

		// IsAPIKeyValid should return false
		isValid := xaiClient.IsAPIKeyValid(ctx)
		assert.False(t, isValid, "Invalid API key should return false")

		// Health check should show failure
		health := xaiClient.HealthCheck(ctx)
		assert.Equal(t, "failed", health["overall_status"])
		assert.Equal(t, "invalid", health["api_key_status"])

		t.Logf("Invalid key test completed: %v", health["api_key_error"])
	})

	t.Run("empty API key", func(t *testing.T) {
		// Create provider with empty API key
		provider, err := NewProvider(
			models.ProviderXAI,
			WithAPIKey(""),
			WithModel(models.SupportedModels[models.XAIGrok3Fast]),
		)
		require.NoError(t, err)

		baseProvider, ok := provider.(*baseProvider[XAIClient])
		require.True(t, ok, "Provider should be baseProvider[XAIClient]")
		xaiClient := baseProvider.client.(*xaiClient)

		ctx := context.Background()

		// Should fail validation
		isValid := xaiClient.IsAPIKeyValid(ctx)
		assert.False(t, isValid, "Empty API key should be invalid")
	})
}
