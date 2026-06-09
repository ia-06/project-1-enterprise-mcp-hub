package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

// WipeProjectIssues deletes all issues in the configured project.
func (a *JiraAdapter) WipeProjectIssues(ctx context.Context) error {
	if a.cfg.JiraProjectKey == "" {
		return nil
	}
	jql := fmt.Sprintf("project = \"%s\"", a.cfg.JiraProjectKey)
	issues, err := a.execJQL(ctx, jql, "")
	if err != nil {
		return err
	}

	for _, issue := range issues {
		endpoint := strings.TrimRight(a.cfg.JiraBaseURL, "/") + "/rest/api/3/issue/" + issue.Key
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
		if err != nil {
			continue
		}
		a.setHeaders(req)
		resp, err := a.client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}
	return nil
}

// SeedIssue creates a new issue of type Task and transitions it to a random workflow state.
func (a *JiraAdapter) SeedIssue(ctx context.Context, accountSFID, summary, description string) {
	if a.cfg.JiraUseMock {
		return
	}

	fieldKey := a.resolveFieldKey(ctx)

	// Create Issue Payload
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": a.cfg.JiraProjectKey,
			},
			"summary":   summary,
			"issuetype": map[string]string{"name": "Task"},
		},
	}

	if fieldKey != "" {
		fields := payload["fields"].(map[string]interface{})
		fields[fieldKey] = accountSFID
	}

	bodyBytes, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(a.cfg.JiraBaseURL, "/") + "/rest/api/3/issue"
	
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[JiraSeed] Error building request: %v", err)
		return
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[JiraSeed] API request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("[JiraSeed] Failed to create issue: %s", string(b))
		return
	}

	var createResp struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return
	}

	// Transition to a random state
	a.transitionToRandomState(ctx, createResp.ID)
}

func (a *JiraAdapter) transitionToRandomState(ctx context.Context, issueID string) {
	// Status workflow: Todo -> In Progress -> Build Broken | Building -> Done -> Reopened
	states := []string{"Todo", "In Progress", "Build Broken", "Building", "Done", "Reopened"}
	targetStatus := states[rand.Intn(len(states))]

	if targetStatus == "Todo" {
		return // default state
	}

	// Fetch available transitions
	endpoint := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", strings.TrimRight(a.cfg.JiraBaseURL, "/"), issueID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var transResp struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&transResp); err != nil {
		return
	}

	// Find matching transition
	var transitionID string
	for _, t := range transResp.Transitions {
		if strings.EqualFold(t.To.Name, targetStatus) || strings.EqualFold(t.Name, targetStatus) {
			transitionID = t.ID
			break
		}
	}

	if transitionID == "" {
		// Just pick a random available transition to simulate movement
		if len(transResp.Transitions) > 0 {
			transitionID = transResp.Transitions[rand.Intn(len(transResp.Transitions))].ID
		} else {
			return
		}
	}

	// POST transition
	payload := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	tReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}
	a.setHeaders(tReq)

	tResp, err := a.client.Do(tReq)
	if err == nil {
		tResp.Body.Close()
	}
}
