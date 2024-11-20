package acode

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xyproto/projectinfo"
)

var psep = string(filepath.Separator)

// TODO: Use accurate token counting instead
var PromptMargin = 1.2 // to compensate for inaccurate token count, this increases it with 20%

func FileContents(project *projectinfo.ProjectInfo, filename string) string {
	return projectinfo.FindFileName(project.ConfAndDocFiles, filename).Contents
}

// ProcessChunk processes a chunk of source code with either the initial or the correction prompt (if not blank)
func (cfg *Config) ProcessChunk(status io.Writer, i, n int, project *projectinfo.ProjectInfo, jsonChunk, promptTemplate, previousAIAnswer string) (string, float64, error) {
	promptData := TemplateData{
		ReadmeContents:   "\n\n" + FileContents(project, "README.md") + "\n",
		SourceCode:       "\n\n" + jsonChunk + "\n",
		PreviousAIAnswer: "\n\n" + previousAIAnswer + "\n",
	}
	prompt, err := cfg.BuildPrompt(promptTemplate, promptData)
	if err != nil {
		return "", 0, err
	}

	// Calculating token count and cost
	sentTokenCount := cfg.CountPromptTokens(prompt)
	if sentTokenCount > cfg.Model.MaxTokens {
		fmt.Fprintf(status, "warning: prompt exceeds approximate token limit: %d tokens (max: %d)\n", sentTokenCount, cfg.Model.MaxTokens)
		if !cfg.Silent {
			log.Printf("warning: prompt exceeds approximate token limit: %d tokens (max: %d)\n", sentTokenCount, cfg.Model.MaxTokens)
		}
		//return "", 0, fmt.Errorf("prompt exceeds the token limit of %d tokens, got %d tokens", cfg.Model.MaxTokens, sentTokenCount)
	}

	result, err := cfg.PostPrompt(prompt)
	if err != nil {
		// Try again, using the fallback model
		tmp := cfg.Model
		cfg.Model = cfg.FallbackModel

		fmt.Fprintf(status, "Error posting prompt (retrying with %s): %v\n", cfg.Model.Name, err)
		if !cfg.Silent {
			log.Printf("Error posting prompt (retrying with %s): %v\n", cfg.Model.Name, err)
		}

		result, err = cfg.PostPrompt(prompt)

		cfg.Model = tmp
	}

	receivedTokenCount := cfg.CountPromptTokens(result)

	usdCost := cfg.Model.CalculateCost(sentTokenCount, receivedTokenCount)

	if n == 1 {
		fmt.Fprintf(status, "Approximate cost: $%.2f for %d sent and %d received tokens.\n", usdCost, sentTokenCount, receivedTokenCount)
	} else {
		fmt.Fprintf(status, "[source code chunk %d/%d] Approximate cost: $%.2f for %d sent and %d received tokens.\n", i+1, n, usdCost, sentTokenCount, receivedTokenCount)
	}

	return result, usdCost, err
}

// processWithPrompt processes the source code JSON chunks with a given prompt and an optional previousAIAnswer string (can be empty)
// it returns a slice of answers and an approximate cost in USD
func (cfg *Config) processWithPrompt(status io.Writer, project *projectinfo.ProjectInfo, jsonChunks []string, prompt, previousAIAnswer string) ([]string, float64) {
	var (
		totalUSDCost float64
		responses    []string
		n            = len(jsonChunks)
	)
	for i := 0; i < n; i++ {
		chunk := jsonChunks[i]
		// Replace the (potentially cryptic temp directory) in the JSON chunk with a blank string
		// but only for a minimum amount of path separators.
		if strings.Count(cfg.Directory, psep) > 2 {
			if !strings.HasSuffix(cfg.Directory, psep) {
				cfg.Directory += psep
			}
			chunk = strings.ReplaceAll(chunk, cfg.Directory, "")
		}
		fmt.Fprintf(status, "Processing chunk %d of %d...\n", i+1, n)
		if !cfg.Silent {
			log.Printf("Processing chunk %d of %d....\n", i+1, n)
		}
		// First process the initial prompt, and with no previous answer
		initialResponse, usdCost, err := cfg.ProcessChunk(status, i, n, project, chunk, prompt, previousAIAnswer)
		if err != nil {
			fmt.Fprintf(status, "Warning processing chunk %d/%d: %v\n", i+1, n, err)
			if !cfg.Silent {
				log.Printf("Warning processing chunk %d/%d: %v\n", i+1, n, err)
			}
			continue
		}
		totalUSDCost += usdCost
		responses = append(responses, strings.TrimSpace(initialResponse))
	}
	return responses, totalUSDCost
}

// Process processes the entire project with AI
// returns the combined initial results, the combined fix results, the confidence from 1 to 10, the cost in USD and an error if applicable
func (cfg *Config) Process(status io.Writer, project *projectinfo.ProjectInfo) (string, string, int, float64, error) {
	var (
		responses, jsonChunks []string
		usdCost, totalUSDCost float64
		err                   error
	)

	fmt.Fprintf(status, "Processing project: %s\n", project.Name)
	if !cfg.Silent {
		log.Printf("Processing project: %s\n", project.Name)
	}

	// First create the prompt without the JSON chunk, then count the tokens and extract that from the MaxToken when chunking

	initialPromptData := TemplateData{
		ReadmeContents:   "\n\n" + FileContents(project, "README.md") + "\n",
		SourceCode:       "\n\n\n",
		PreviousAIAnswer: "\n\n\n",
	}
	promptWithoutSourceCode, err := cfg.BuildPrompt(cfg.InitialPrompt, initialPromptData)
	if err != nil {
		return "", "", 0, 0, err
	}
	barePromptTokenCount := cfg.CountPromptTokens(promptWithoutSourceCode)

	// TODO: Let project.Chunk take an extra barePromptTokenCount int
	cfg.Model.MaxTokens -= int(float64(barePromptTokenCount) * PromptMargin)
	jsonChunks, err = Chunk(cfg, project, !cfg.ExcludeSources, cfg.IncludeConfAndDoc)
	if err != nil {
		cfg.Model.MaxTokens += barePromptTokenCount
		return "", "", 0, 0, err
	}
	cfg.Model.MaxTokens += barePromptTokenCount

	fmt.Fprintf(status, "Project chunked into %d chunks.\n", len(jsonChunks))
	if !cfg.Silent {
		log.Printf("Project chunked into %d chunks.\n", len(jsonChunks))
	}

	fmt.Fprintln(status, "Using the initial prompt...")
	if !cfg.Silent {
		log.Println("Using the initial prompt...")
	}

	// Process the chunks with the initial prompt, and prepare to return combinedInitialResponses
	responses, usdCost = cfg.processWithPrompt(status, project, jsonChunks, cfg.InitialPrompt, "")
	totalUSDCost += usdCost
	combinedInitialResponses := ""
	for _, response := range responses {
		if strings.HasPrefix(response, "No") && strings.HasSuffix(response, "found.") {
			continue
		}
		combinedInitialResponses += "\n" + response
	}
	combinedInitialResponses = strings.TrimSpace(combinedInitialResponses)

	nothingFound := combinedInitialResponses == "" || (strings.HasPrefix(combinedInitialResponses, "No ") && strings.Count(combinedInitialResponses, " ") < 5)

	var (
		combinedFixResponses string
		confidence           int
	)

	if cfg.AlsoOutputFixAndConfidence && !nothingFound {
		fmt.Fprintln(status, "Using the prompt that finds fixes...")
		if !cfg.Silent {
			log.Println("Using the prompt that finds fixes...")
		}

		// Process the chunks with the fix prompt, and prepare to return combinedFixResponses
		responses, usdCost = cfg.processWithPrompt(status, project, jsonChunks, cfg.FixPrompt, combinedInitialResponses)
		totalUSDCost += usdCost
		for _, response := range responses {
			if strings.HasPrefix(response, "No ") && (strings.HasSuffix(response, " found.") || strings.HasSuffix(response, " needed.")) {
				continue
			}
			combinedFixResponses += "\n" + response
		}

		fmt.Fprintln(status, "Using the prompt that judges confidence...")
		if !cfg.Silent {
			log.Println("Using the prompt that judges confidence...")
		}

		// Process the chunks with the confidence prompt, and prepare to return combinedConfidenceResponses
		responses, usdCost = cfg.processWithPrompt(status, project, jsonChunks, cfg.ConfidencePrompt, combinedInitialResponses)
		totalUSDCost += usdCost
		var found bool
		var confidenceFloat float64 = 5.0 // from 1 to 10, will be converted to an int before it is returned
		for _, response := range responses {
			if n, err := strconv.Atoi(response); err == nil { // success
				if !found {
					confidenceFloat = float64(n)
					found = true
				} else {
					confidenceFloat = (confidenceFloat + float64(n)) / 2.0
				}
			}
		}
		confidence = int(confidenceFloat)
	}

	combinedInitialResponses = strings.TrimSpace(combinedInitialResponses)
	if combinedInitialResponses == "" {
		switch cfg.OpType {
		case OpGenAPI:
			combinedInitialResponses = "No documentation generated."
		case OpGenReadme:
			combinedInitialResponses = "No documentation generated."
		case OpGenCatalog:
			combinedInitialResponses = "No configuration generated."
		case OpGenAnyFile:
			combinedInitialResponses = "No file generated."
		case OpFindBug:
			combinedInitialResponses = "No bugs found."
		case OpFindTypo:
			combinedInitialResponses = "No typos found."
		case OpGenDoc:
			fallthrough
		default:
			combinedInitialResponses = "No documentation generated."
		}
	}

	return combinedInitialResponses, combinedFixResponses, confidence, totalUSDCost, nil
}
