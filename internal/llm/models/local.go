package models

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/spf13/viper"
)

const (
	ProviderLocal ModelProvider = "local"

	localModelsPath        = "v1/models"
	lmStudioBetaModelsPath = "api/v0/models"
)

func initLocalModels() {
	if endpoint := os.Getenv("LOCAL_ENDPOINT"); endpoint != "" {
		localEndpoint, err := url.Parse(endpoint)
		if err != nil {
			logging.Debug("Failed to parse local endpoint",
				"error", err,
				"endpoint", endpoint,
			)
			return
		}

		load := func(url *url.URL, path string) []localModel {
			url = url.JoinPath(path)
			logging.Debug(fmt.Sprintf("Trying to load models from %s", url))
			return listLocalModels(url.String())
		}

		models := load(localEndpoint, lmStudioBetaModelsPath)

		if len(models) == 0 {
			models = load(localEndpoint, localModelsPath)
		}

		if c := len(models); c == 0 {
			logging.Debug("No local models found",
				"endpoint", endpoint,
			)
			return
		} else {
			logging.Debug(fmt.Sprintf("%d local models found", c))
		}

		loadLocalModels(models)

		if token, ok := os.LookupEnv("LOCAL_ENDPOINT_API_KEY"); ok {
			viper.SetDefault("providers.local.apiKey", token)
		} else {
			viper.SetDefault("providers.local.apiKey", "dummy")
		}
		ProviderPopularity[ProviderLocal] = 0
	}
}

type localModelList struct {
	Data []localModel `json:"data"`
}

type localModel struct {
	ID                  string `json:"id"`
	Object              string `json:"object"`
	Type                string `json:"type"`
	Publisher           string `json:"publisher"`
	Arch                string `json:"arch"`
	CompatibilityType   string `json:"compatibility_type"`
	Quantization        string `json:"quantization"`
	State               string `json:"state"`
	MaxContextLength    int64  `json:"max_context_length"`
	LoadedContextLength int64  `json:"loaded_context_length"`
}

func listLocalModels(modelsEndpoint string) []localModel {
	token := os.Getenv("LOCAL_ENDPOINT_API_KEY")
	var (
		res *http.Response
		err error
	)
	if token != "" {
		req, reqErr := http.NewRequest("GET", modelsEndpoint, nil)
		if reqErr != nil {
			logging.Debug("Failed to create local models request",
				"error", reqErr,
				"endpoint", modelsEndpoint,
			)
			return nil
		}
		req.Header.Set("Authorization", "Bearer "+token)
		res, err = http.DefaultClient.Do(req)
	} else {
		res, err = http.Get(modelsEndpoint)
	}
	if err != nil || res == nil {
		logging.Debug("Failed to list local models",
			"error", err,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		logging.Debug("Failed to list local models",
			"status", res.StatusCode,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}

	var modelList localModelList
	if err = json.NewDecoder(res.Body).Decode(&modelList); err != nil {
		logging.Debug("Failed to list local models",
			"error", err,
			"endpoint", modelsEndpoint,
		)
		return []localModel{}
	}

	var supportedModels []localModel
	for _, model := range modelList.Data {
		if strings.HasSuffix(modelsEndpoint, lmStudioBetaModelsPath) {
			if model.Object != "model" || model.Type != "llm" {
				logging.Debug("Skipping unsupported LMStudio model",
					"endpoint", modelsEndpoint,
					"id", model.ID,
					"object", model.Object,
					"type", model.Type,
				)

				continue
			}
		}

		supportedModels = append(supportedModels, model)
	}

	return supportedModels
}

func loadLocalModels(models []localModel) {
	for i, m := range models {
		source := tryResolveSource(m.ID)
		model := convertLocalModel(m, source)
		SupportedModels[model.ID] = model

		if i == 0 || m.State == "loaded" {
			viper.SetDefault("agents.coder.model", model.ID)
			viper.SetDefault("agents.summarizer.model", model.ID)
			viper.SetDefault("agents.task.model", model.ID)
			viper.SetDefault("agents.title.model", model.ID)
		}
	}
}

func tryResolveSource(localID string) *Model {
	for _, model := range SupportedModels {
		if strings.Contains(localID, model.APIModel) {
			return &model
		}
	}
	return nil
}

func convertLocalModel(model localModel, source *Model) Model {
	if source != nil {
		m := *source
		m.ID = ModelID("local." + model.ID)
		m.Name = source.Name
		m.APIModel = model.ID
		m.Provider = ProviderLocal
		return m
	} else {
		return Model{
			ID:                  ModelID("local." + model.ID),
			Name:                friendlyModelName(model.ID),
			Provider:            ProviderLocal,
			APIModel:            model.ID,
			ContextWindow:       cmp.Or(model.LoadedContextLength, 4096),
			DefaultMaxTokens:    cmp.Or(model.LoadedContextLength, 4096),
			CanReason:           false,
			SupportsAttachments: false,
		}
	}
}

var modelInfoRegex = regexp.MustCompile(`(?i)^([a-z0-9]+)(?:[-_]?([rv]?\d[\.\d]*))?(?:[-_]?([a-z]+))?.*`)

func friendlyModelName(modelID string) string {
	mainID := modelID
	tag := ""

	if slash := strings.LastIndex(mainID, "/"); slash != -1 {
		mainID = mainID[slash+1:]
	}

	if at := strings.Index(modelID, "@"); at != -1 {
		mainID = modelID[:at]
		tag = modelID[at+1:]
	}

	match := modelInfoRegex.FindStringSubmatch(mainID)
	if match == nil {
		return modelID
	}

	capitalize := func(s string) string {
		if s == "" {
			return ""
		}
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	}

	family := capitalize(match[1])
	version := ""
	label := ""

	if len(match) > 2 && match[2] != "" {
		version = strings.ToUpper(match[2])
	}

	if len(match) > 3 && match[3] != "" {
		label = capitalize(match[3])
	}

	var parts []string
	if family != "" {
		parts = append(parts, family)
	}
	if version != "" {
		parts = append(parts, version)
	}
	if label != "" {
		parts = append(parts, label)
	}
	if tag != "" {
		parts = append(parts, tag)
	}

	return strings.Join(parts, " ")
}
