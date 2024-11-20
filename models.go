package acode

type Model struct {
	Description                              string
	Name                                     string
	MaxTokens                                int
	PostURL                                  string
	USDPerMillionTokensForShortPrompts       float64 // < 128K tokens
	USDPerMillionTokensForLongPrompts        float64 // > 128K tokens
	USDPerMillionTokensOutputForShortPrompts float64 // < 128K tokens
	USDPerMillionTokensOutputForLongPrompts  float64 // > 128K tokens
}

// AllModels holds the list of models configured by the caller.
var AllModels []Model

// SetModels allows the caller to configure the available models.
func SetModels(models []Model) {
	AllModels = models
}

func AllModelNames() []string {
	var xs []string
	for _, m := range AllModels {
		xs = append(xs, m.Name)
	}
	return xs
}

func ModelNamesAndDescriptions() map[string]string {
	nameDescMap := make(map[string]string, len(AllModels))
	for _, m := range AllModels {
		nameDescMap[m.Name] = m.Description
	}
	return nameDescMap
}

// CalculateCostFromStrings returns the approximate cost in USD
func (model *Model) CalculateCostFromStrings(cfg *Config, inputString, outputString string) float64 {
	return model.CalculateCost(cfg.CountPromptTokens(inputString), cfg.CountPromptTokens(outputString))
}

// CalculateCost returns the approximate cost in USD
func (model *Model) CalculateCost(sentTokenCount, receivedTokenCount int) float64 {
	usdPerInputToken := model.USDPerMillionTokensForShortPrompts / 1000000.0
	usdPerOutputToken := model.USDPerMillionTokensOutputForShortPrompts / 1000000.0
	if sentTokenCount > 128000 {
		usdPerInputToken = model.USDPerMillionTokensForLongPrompts / 1000000.0
		usdPerOutputToken = model.USDPerMillionTokensOutputForLongPrompts / 1000000.0
	}
	return float64(sentTokenCount)*usdPerInputToken + float64(receivedTokenCount)*usdPerOutputToken
}
