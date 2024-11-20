package acode

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/projectinfo"
)

type Config struct {
	Model                      Model
	FallbackModel              Model
	Output                     *os.File
	InitialPrompt              string
	FixPrompt                  string
	ConfidencePrompt           string
	OutputFilename             string
	Force                      bool
	Silent                     bool
	OutputPrompt               bool
	Version                    bool
	Directory                  string
	OpType                     OperationType
	IncludeConfAndDoc          bool
	ExcludeSources             bool
	AlsoOutputFixAndConfidence bool
	Timeout                    time.Duration
}

// NewConfig initializes a new Config with default settings and default prompts
func NewConfig(defaultModel, fallbackModel *Model) *Config {
	var cfg Config
	cfg.Model = *defaultModel
	cfg.FallbackModel = *fallbackModel
	cfg.Model.PostURL = env.Str("POSTURL", cfg.Model.PostURL)
	cfg.Model.Name = env.Str("MODELNAME", cfg.Model.Name)
	cfg.Model.MaxTokens = env.Int("MAXTOKENS", cfg.Model.MaxTokens)
	cfg.Timeout = 2 * time.Minute
	cfg.Directory = "." // the default value
	return &cfg
}

// configureCommonSettings configures common settings for the configuration based on provided arguments and flags.
func (cfg *Config) configureCommonSettings(customInitialPrompt, customFixPrompt, customConfidencePrompt string, opType OperationType) error {

	cfg.OpType = opType

	switch cfg.OpType {
	case OpGenReadme, OpGenAPI, OpGenDoc, OpGenAnyFile:
		cfg.IncludeConfAndDoc = true
	case OpGenCatalog:
		cfg.IncludeConfAndDoc = true
		cfg.ExcludeSources = true
	}

	// Set custom or default prompts
	if customInitialPrompt != "" {
		cfg.InitialPrompt = customInitialPrompt
	} else {
		cfg.InitialPrompt = GetInitialPrompt(cfg.OpType)
	}

	if customFixPrompt != "" {
		cfg.FixPrompt = customFixPrompt
	} else {
		cfg.FixPrompt = GetFixPrompt(cfg.OpType)
	}

	if customConfidencePrompt != "" {
		cfg.ConfidencePrompt = customConfidencePrompt
	} else {
		cfg.ConfidencePrompt = GetConfidencePrompt(cfg.OpType)
	}

	return nil
}

func (cfg *Config) GatherSources(customInitialPrompt, customFixPrompt, customConfidencePrompt string, opType OperationType) (*projectinfo.ProjectInfo, error) {
	err := cfg.configureCommonSettings(customInitialPrompt, customFixPrompt, customConfidencePrompt, opType)
	if err != nil {
		return nil, err
	}

	inputDirectory := cfg.Directory

	if ok, err := files.DirectoryWithFiles(inputDirectory); err != nil {
		return nil, fmt.Errorf("ReactToConfigAndReadSources: error examining the input directory: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("ReactToConfigAndReadSources: the given directory %s does not contain at least one file", inputDirectory)
	}

	printWarnings := true
	project, err := projectinfo.New(inputDirectory, printWarnings)
	if err != nil {
		return nil, err
	}

	if cfg.OpType == 0 {
		if project.APIServer {
			cfg.OpType = OpGenAPI
		}
	}

	return &project, nil
}

// ReactToConfigAndReadSources sets up command line flags for the application
func (cfg *Config) ReactToConfigAndReadSources(args []string, customInitialPrompt, customFixPrompt, customConfidencePrompt string, apidoc, bug, catalog, readme, typo bool) (*projectinfo.ProjectInfo, error) {
	inputDirectory := "."
	if args := args; len(args) > 0 {
		inputDirectory = args[0]
	}

	if ok, err := files.DirectoryWithFiles(inputDirectory); err != nil {
		return nil, fmt.Errorf("ReactToConfigAndReadSources: error examining the input directory: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("ReactToConfigAndReadSources: the given directory %s does not contain at least one file", inputDirectory)
	}

	cfg.Directory = inputDirectory

	var opType OperationType = OpGenDoc
	if bug {
		opType = OpFindBug
	} else if typo {
		opType = OpFindTypo
	} else if readme {
		opType = OpGenReadme
	} else if catalog {
		opType = OpGenCatalog
	} else if apidoc {
		opType = OpGenAPI
	} // OpGenAnyFile is not added here yet, on purpose

	err := cfg.configureCommonSettings(customInitialPrompt, customFixPrompt, customConfidencePrompt, opType)
	if err != nil {
		return nil, err
	}

	const printWarnings = true
	project, err := projectinfo.New(cfg.Directory, printWarnings)
	if err != nil {
		return nil, err
	}

	if project.Name == "" {
		project.Name = filepath.Base(inputDirectory)
	}

	if cfg.OpType == 0 {
		if project.APIServer {
			cfg.OpType = OpGenAPI
		}
	}

	if cfg.OutputFilename == "" {
		// Set output file based on the operation type
		cfg.OutputFilename = GetDefaultFilename(cfg.OpType)
	}

	return &project, nil
}

// InitializeOutputFile opens the output file based on configuration
func (cfg *Config) InitializeOutputFile() {
	if cfg.OutputFilename == "-" || cfg.OutputFilename == "" {
		cfg.Output = os.Stdout
	} else {
		var err error
		cfg.Output, err = os.OpenFile(cfg.OutputFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open output file %s: %v\n", cfg.OutputFilename, err)
			os.Exit(1)
		}
	}
}

// BuildPrompt constructs the final prompt from the template and data
func (cfg *Config) BuildPrompt(promptTemplate string, templateData TemplateData) (string, error) {
	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return result.String(), nil
}
