package acode

import (
	"encoding/json"
	"fmt"

	"github.com/xyproto/projectinfo"
)

// Chunk breaks down project information into manageable JSON chunks to adhere to token limitations
func Chunk(cfg *Config, project *projectinfo.ProjectInfo, includeSourceFiles, includeConfAndDocFiles bool) ([]string, error) {
	var (
		currentTokenCount int
		chunks            []string
		currentChunk      []projectinfo.FileInfo
		files             = []projectinfo.FileInfo{}
	)
	if includeSourceFiles {
		files = append(files, project.SourceFiles...)
	}
	if includeConfAndDocFiles {
		files = append(files, project.ConfAndDocFiles...)
	}
	for _, file := range files {
		file.TokenCount = cfg.CountPromptTokens(file.Contents) // Compute token count for each file, assuming this function is defined in utils.go.
		if currentTokenCount+file.TokenCount > cfg.Model.MaxTokens {
			// Finalize the current chunk and reset counters if the maximum token count is exceeded.
			chunkData, err := json.Marshal(currentChunk)
			if err != nil {
				return nil, fmt.Errorf("error marshaling chunk: %v", err)
			}
			chunks = append(chunks, string(chunkData))
			currentChunk = []projectinfo.FileInfo{} // Reset the current chunk
			currentTokenCount = 0
		}
		// Add the file to the current chunk
		currentChunk = append(currentChunk, file)
		currentTokenCount += file.TokenCount
	}
	// Add the last chunk if it contains any files
	if len(currentChunk) > 0 {
		chunkData, err := json.Marshal(currentChunk)
		if err != nil {
			return nil, fmt.Errorf("error marshaling final chunk: %v", err)
		}
		chunks = append(chunks, string(chunkData))
	}
	return chunks, nil
}
