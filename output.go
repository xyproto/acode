package acode

import (
	"fmt"
	"os"

	"github.com/xyproto/ask"
	"github.com/xyproto/files"
)

func (cfg *Config) OutputResponse(response string) error {
	if cfg.OutputFilename == "-" || cfg.OutputFilename == "" {
		fmt.Println(response)
		return nil
	}

	if !cfg.Force && files.Exists(cfg.OutputFilename) && !ask.YN(cfg.OutputFilename+" already exists. Overwrite it?") {
		fmt.Println("Did nothing.")
		return nil
	}

	if err := os.WriteFile(cfg.OutputFilename, []byte(response), 0644); err != nil {
		return fmt.Errorf("failed to write to output file %s: %v", cfg.OutputFilename, err)
	}

	if !cfg.Silent {
		fmt.Println("Output written successfully to", cfg.OutputFilename)
	}
	return nil
}
