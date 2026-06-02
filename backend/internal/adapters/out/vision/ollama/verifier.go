package ollama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pokemon-binder-finder/internal/domain"
)

type Verifier struct {
	baseURL string
	model   string
	client  *http.Client
}

func New(baseURL, model string) *Verifier {
	return &Verifier{baseURL: strings.TrimRight(baseURL, "/"), model: model, client: &http.Client{Timeout: 90 * time.Second}}
}

func (v *Verifier) Available(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/api/tags", nil)
	resp, err := v.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return false
}

func (v *Verifier) Verify(ctx context.Context, reference, candidate []byte, match domain.MatchCandidate) (float64, string, error) {
	payload := map[string]any{
		"model":  v.model,
		"stream": false,
		"format": "json",
		"messages": []map[string]any{{
			"role": "user",
			"content": `Compare these Pokemon card images. Return JSON only: {"confidence": number, "reason": string}. ` +
				"Use confidence from 0 to 1 for whether the second image contains the exact printing shown in the first image.",
			"images": []string{base64.StdEncoding.EncodeToString(reference), base64.StdEncoding.EncodeToString(candidate)},
		}},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return match.Confidence, match.Reason, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return match.Confidence, match.Reason, fmt.Errorf("ollama returned %s", resp.Status)
	}
	var response struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return match.Confidence, match.Reason, err
	}
	var verdict struct {
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(response.Message.Content), &verdict); err != nil {
		return match.Confidence, match.Reason, err
	}
	adjusted := match.Confidence + (verdict.Confidence-0.5)*0.20
	if adjusted < 0 {
		adjusted = 0
	}
	if adjusted > 1 {
		adjusted = 1
	}
	return adjusted, match.Reason + "; ollama: " + verdict.Reason, nil
}
