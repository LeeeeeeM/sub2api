package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/typesafe"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

// ListAccountConnectivityModelsPayload returns the same model list shape as
// GET /api/v1/admin/accounts/:id/models (account connectivity / test picker).
func (s *AccountTestService) ListAccountConnectivityModelsPayload(ctx context.Context, account *Account) any {
	if account == nil {
		return []claude.Model{}
	}

	if account.IsOpenAI() {
		if s != nil {
			if models, fetchErr := s.FetchOpenAIAccountModels(ctx, account); fetchErr == nil {
				return models
			}
		}
		if account.IsOpenAIPassthroughEnabled() {
			return openai.DefaultModels
		}
		mapping := account.GetModelMapping()
		if len(mapping) == 0 {
			return openai.DefaultModels
		}
		var models []openai.Model
		for requestedModel := range mapping {
			var found bool
			for _, dm := range openai.DefaultModels {
				if dm.ID == requestedModel {
					models = append(models, dm)
					found = true
					break
				}
			}
			if !found {
				models = append(models, openai.Model{
					ID:          requestedModel,
					Object:      "model",
					Type:        "model",
					DisplayName: requestedModel,
				})
			}
		}
		return models
	}

	if account.IsGemini() {
		if account.IsOAuth() {
			if account.IsGeminiGoogleOne() {
				return geminicli.GoogleOneModels
			}
			return geminicli.DefaultModels
		}
		mapping := account.GetModelMapping()
		if len(mapping) == 0 {
			return geminicli.DefaultModels
		}
		var models []geminicli.Model
		for requestedModel := range mapping {
			var found bool
			for _, dm := range geminicli.DefaultModels {
				if dm.ID == requestedModel {
					models = append(models, dm)
					found = true
					break
				}
			}
			if !found {
				models = append(models, geminicli.Model{
					ID:          requestedModel,
					Type:        "model",
					DisplayName: requestedModel,
					CreatedAt:   "",
				})
			}
		}
		return models
	}

	if account.Platform == PlatformAntigravity {
		return antigravityAccountConnectivityModels(account.Credentials["model_mapping"])
	}

	if account.Platform == PlatformGrok {
		defaultModels := xai.DefaultModels()
		hasExplicitMapping := false
		switch rawMapping := account.Credentials["model_mapping"].(type) {
		case map[string]any:
			hasExplicitMapping = len(rawMapping) > 0
		case map[string]string:
			hasExplicitMapping = len(rawMapping) > 0
		}
		if !hasExplicitMapping {
			return defaultModels
		}
		mapping := account.GetModelMapping()
		if len(mapping) == 0 {
			return defaultModels
		}
		defaultByID := make(map[string]xai.Model, len(defaultModels))
		for _, model := range defaultModels {
			defaultByID[model.ID] = model
		}
		requestedModels := make([]string, 0, len(mapping))
		for requestedModel := range mapping {
			requestedModels = append(requestedModels, requestedModel)
		}
		sort.Strings(requestedModels)
		var models []xai.Model
		for _, requestedModel := range requestedModels {
			if defaultModel, found := defaultByID[requestedModel]; found {
				models = append(models, defaultModel)
				continue
			}
			models = append(models, xai.Model{
				ID:          requestedModel,
				Object:      "model",
				OwnedBy:     "xai",
				DisplayName: requestedModel,
			})
		}
		return models
	}

	if account.IsTypeSafe() {
		return []claude.Model{{ID: typesafe.JevLatestModel, Type: "model", DisplayName: typesafe.JevLatestModel}}
	}

	if account.IsOAuth() {
		return claude.DefaultModels
	}

	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return claude.DefaultModels
	}
	var models []claude.Model
	for requestedModel := range mapping {
		var found bool
		for _, dm := range claude.DefaultModels {
			if dm.ID == requestedModel {
				models = append(models, dm)
				found = true
				break
			}
		}
		if !found {
			models = append(models, claude.Model{
				ID:          requestedModel,
				Type:        "model",
				DisplayName: requestedModel,
				CreatedAt:   "",
			})
		}
	}
	return models
}

// ListAccountConnectivityModelIDs extracts public model IDs from the admin
// connectivity model payload.
func (s *AccountTestService) ListAccountConnectivityModelIDs(ctx context.Context, account *Account) []string {
	return ConnectivityModelIDsFromPayload(s.ListAccountConnectivityModelsPayload(ctx, account))
}

// ConnectivityModelIDsFromPayload collects model IDs from admin connectivity payloads.
func ConnectivityModelIDsFromPayload(payload any) []string {
	switch models := payload.(type) {
	case []openai.Model:
		return modelIDsFromStrings(openaiModelIDs(models))
	case []geminicli.Model:
		return modelIDsFromStrings(geminicliModelIDs(models))
	case []claude.Model:
		return modelIDsFromStrings(claudeModelIDs(models))
	case []xai.Model:
		return modelIDsFromStrings(xaiModelIDs(models))
	case []antigravity.ClaudeModel:
		return modelIDsFromStrings(antigravityModelIDs(models))
	default:
		return nil
	}
}

func modelIDsFromStrings(ids []string) []string {
	if len(ids) == 0 {
		return ids
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func openaiModelIDs(models []openai.Model) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func geminicliModelIDs(models []geminicli.Model) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func claudeModelIDs(models []claude.Model) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func xaiModelIDs(models []xai.Model) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func antigravityModelIDs(models []antigravity.ClaudeModel) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

// antigravityAccountConnectivityModels uses the stored mapping rather than GetModelMapping,
// which supplies defaults and compatibility aliases that the administrator did not configure.
func antigravityAccountConnectivityModels(rawMapping any) []antigravity.ClaudeModel {
	var mappedIDs []string
	switch mapping := rawMapping.(type) {
	case map[string]any:
		for id := range mapping {
			if strings.TrimSpace(id) != "" {
				mappedIDs = append(mappedIDs, id)
			}
		}
	case map[string]string:
		for id := range mapping {
			if strings.TrimSpace(id) != "" {
				mappedIDs = append(mappedIDs, id)
			}
		}
	}
	if len(mappedIDs) == 0 {
		return antigravity.DefaultModels()
	}

	sort.Strings(mappedIDs)
	defaultByID := make(map[string]antigravity.ClaudeModel)
	for _, model := range antigravity.DefaultModels() {
		defaultByID[model.ID] = model
	}
	models := make([]antigravity.ClaudeModel, 0, len(mappedIDs))
	for _, id := range mappedIDs {
		if model, ok := defaultByID[id]; ok {
			models = append(models, model)
			continue
		}
		models = append(models, antigravity.ClaudeModel{
			ID:          id,
			Type:        "model",
			DisplayName: id,
		})
	}
	return models
}
