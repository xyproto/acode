package acode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/xyproto/projectinfo"
)

// trimCodeBlockMarkers removes leading "```" or "```yaml" and trailing "```" from the string.
func trimCodeBlockMarkers(input string) string {
	re := regexp.MustCompile(`(?ms)^\s*\x60{3}(?:[a-zA-Z]+)?\n(.*?)\n\x60{3}$`)
	if matches := re.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}
	return input
}

// PostPrompt sends the given prompt and model name to the configured AI server and returns the answer.
// The answer may optionally be trimmed for code block markers (ie. ```yaml ... ```).
func (cfg *Config) PostPrompt(prompt string) (string, error) {
	var (
		requestBody []byte
		err         error
	)

	// Prepare the JSON payload for POST request
	if strings.HasPrefix(cfg.Model.Name, "gemini") {
		requestBody, err = json.Marshal(map[string]interface{}{
			"prompt":      prompt,
			"model":       cfg.Model.Name,
			"temperature": 0, // generating documentation should not be too creative
		})
	} else {
		requestBody, err = json.Marshal(map[string]interface{}{
			"prompt": prompt,
		})
	}

	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Create and send the POST request
	req, err := http.NewRequestWithContext(ctx, "POST", cfg.Model.PostURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	if !cfg.Silent {
		log.Printf("Sending a request to %s using the %s model... \n", cfg.Model.PostURL, cfg.Model.Name)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read and process the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if !cfg.Silent {
		log.Printf("Recevied %d bytes from the server.\n", len(responseBody))
	}

	var response map[string]string

	if err := json.Unmarshal(responseBody, &response); err == nil { // success
		if answer, exists := response["answer"]; exists { // success
			if cfg.OpType == OpGenCatalog || cfg.OpType == OpGenAnyFile {
				return trimCodeBlockMarkers(answer), nil
			}
			return answer, nil
		}
	}

	responseString := string(responseBody)

	if strings.Contains(responseString, "</title>403 Forbidden") {
		return "", fmt.Errorf("got %q when contacting %s, are the network settings correct?", "403 Forbidden", cfg.Model.PostURL)
	}

	return responseString, nil
}

// CountPromptTokens sends the given prompt and model name to the configured AI server and counts the tokens.
// Only works for gemini*, for now. For the other models, the tokens are estimated.
// If there are errors, a warning is logged and the tokens are estimated instead.
func (cfg *Config) CountPromptTokens(prompt string) int {

	PostURL := strings.Replace(cfg.Model.PostURL, "/query", "/counttext", 1)

	var (
		requestBody []byte
		err         error
	)

	// Prepare the JSON payload for POST request
	if strings.HasPrefix(cfg.Model.Name, "gemini") {
		requestBody, err = json.Marshal(map[string]interface{}{
			"prompt": prompt,
			"model":  cfg.Model.Name,
		})
	} else {
		return projectinfo.CountTokens(prompt)
	}

	if err != nil {
		log.Printf("warning: could not marshal request body: %v\n", err)
		return projectinfo.CountTokens(prompt)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Create and send the POST request
	req, err := http.NewRequestWithContext(ctx, "POST", PostURL, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("warning: could not create token count POST request: %v\n", err)
		return projectinfo.CountTokens(prompt)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	if !cfg.Silent {
		log.Printf("Sending a token count request to %s using the %s model... \n", PostURL, cfg.Model.Name)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("warning: count not send token count request: %v\n", err)
		return projectinfo.CountTokens(prompt)
	}
	defer resp.Body.Close()

	// Read and process the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("warning: could not read token count response body: %v\n", err)
		return projectinfo.CountTokens(prompt)
	}

	responseString := string(responseBody)

	// expects only one key "tokens" with only one int value, and nothing else
	var response map[string]int

	if err := json.Unmarshal(responseBody, &response); err == nil { // success
		if tokenCount, exists := response["tokens"]; exists { // success
			return tokenCount
		}
	}

	if strings.Contains(responseString, "</title>403 Forbidden") {
		log.Printf("got %q when contacting %s, are the network settings correct?\n", "403 Forbidden", PostURL)
		return projectinfo.CountTokens(prompt)
	}

	log.Printf("warning: got a string back from %s, expected JSON with a token count instead: %s\n", PostURL, strings.TrimSpace(responseString))
	return projectinfo.CountTokens(prompt)
}
