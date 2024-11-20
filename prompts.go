package acode

// TemplateData defines the structure used for prompt templates
type TemplateData struct {
	ReadmeContents   string
	SourceCode       string
	PreviousAIAnswer string
}

type OperationType int

// The different operations that this program can do
const (
	OpGenDoc     = iota // generate general documentation
	OpGenAPI            // generate API documentation
	OpGenReadme         // generate a README.md style file
	OpGenCatalog        // generate app-catalog.yaml config for Backstage
	OpGenAnyFile        // generate any file
	OpFindBug           // find a bug
	OpFindTypo          // find a typo
)

func GetDefaultFilename(opType OperationType) string {
	switch opType {
	case OpGenAPI:
		return "API.md"
	case OpGenReadme:
		return "README.md"
	case OpGenCatalog:
		return "app-catalog.yaml"
	case OpGenAnyFile:
		return "unspecified.filename"
	case OpFindBug, OpFindTypo:
		return "-"
	case OpGenDoc:
		fallthrough
	default:
		return "DOC.md"
	}
}

func GetInitialPrompt(opType OperationType) string {
	switch opType {
	case OpGenAPI:
		return `Generate detailed and precise API documentation in Markdown format. Focus on the structure, functionality, and usage of the API server code provided. Assume the reader is technically proficient but unfamiliar with this project. Provide only factual information and indicate "TBD" if information is missing. Do not mention being an AI. Ensure accuracy in all details.
{{.SourceCode}}`
	case OpGenReadme:
		return `Create a comprehensive README.md file in Markdown format, serving as the initial contact for developers and users of this project. Include the following sections:
1. Project title and a brief description highlighting its purpose and value.
2. Step-by-step installation instructions, including any required software or dependencies.
{{.ReadmeContents}}
3. Usage instructions with examples.
4. List of main features and functionalities.
5. Contributing guidelines for new developers, covering coding standards, pull requests, and issue filing.
6. License information.
7. Contact details or links for further discussion.
Assume a basic understanding of software projects but unfamiliarity with this specific project or technology stack. Be clear, concise, and factually accurate. Omit sections with insufficient information.
{{.SourceCode}}`
	case OpGenCatalog:
		return `Create a catalog-info.yaml file in YAML format for the Backstage software catalog with the following structure:
1. apiVersion: 'backstage.io/v1alpha1' – Ensure compatibility with the Backstage platform.
2. kind: Determine based on README.md, API.md, and DOC.md files.
3. metadata: Extract the project's name, title, and description from available documentation. Include repository annotations if the URL is available.
4. spec: Categorize the component type, lifecycle status, and ownership information using existing documentation.
Do not make assumptions or introduce inaccuracies.
{{.SourceCode}}`
	case OpFindBug:
		return `Review the following code and identify any bugs. If no bugs are found, respond with "No bugs found." Be certain of any bug before reporting. Prioritize false positives over false negatives. Include the file name if a bug is found.
{{.SourceCode}}`
	case OpFindTypo:
		return `Review the following code for typos in comments. If no typos are found, respond with "No typos found." Be certain of any typo before reporting. Prioritize false positives over false negatives. Include the file name if a typo is found.
{{.SourceCode}}`
	case OpGenDoc:
		return `Create comprehensive software documentation in Markdown format. Provide a clear overview of the architecture, components, and interfaces of the software. Describe each component's responsibilities and interactions. Include code snippets and configurations to enhance understanding.
{{.SourceCode}}`
	default:
		return `Generate a haiku about this source code: {{.SourceCode}}`
	}
}

func GetFixPrompt(opType OperationType) string {
	switch opType {
	case OpGenAPI:
		return `Generate a diff to update or fix the API.md file based on this new API.md file: {{.PreviousAIAnswer}} and this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpGenReadme:
		return `Generate a diff to update or fix the README.md file based on this new README.md file: {{.PreviousAIAnswer}} and this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpGenCatalog:
		return `Generate a diff to update or fix the app-catalog.yaml file based on this new app-catalog.yaml file: {{.PreviousAIAnswer}} and this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpGenAnyFile:
		return `Generate a diff to update or fix the file based on this new file: {{.PreviousAIAnswer}} and this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpFindBug:
		return `Generate a diff to fix these bugs: {{.PreviousAIAnswer}} in this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpFindTypo:
		return `Generate a diff to fix these typos: {{.PreviousAIAnswer}} in this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	case OpGenDoc:
		return `Generate a diff to update or fix the DOC.md file based on this new DOC.md file: {{.PreviousAIAnswer}} and this project source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	default:
		return `Given these findings and/or generated files: {{.PreviousAIAnswer}}, generate a diff to improve this source code: {{.SourceCode}}. If no changes are needed, respond with "No diff needed." Ensure all filenames and details are correct.`
	}
}

func GetConfidencePrompt(opType OperationType) string {
	switch opType {
	case OpGenAPI:
		return `How confident are you that this API documentation: {{.PreviousAIAnswer}} is accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpGenReadme:
		return `How confident are you that this README.md file: {{.PreviousAIAnswer}} is accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpGenCatalog:
		return `How confident are you that this Backstage configuration: {{.PreviousAIAnswer}} is accurate for a project with this source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpGenAnyFile:
		return `How confident are you that this file: {{.PreviousAIAnswer}} is accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpFindBug:
		return `How confident are you that these bug findings: {{.PreviousAIAnswer}} are accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpFindTypo:
		return `How confident are you that these typo findings: {{.PreviousAIAnswer}} are accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	case OpGenDoc:
		return `How confident are you that this documentation: {{.PreviousAIAnswer}} is accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	default:
		return `How confident are you that this result: {{.PreviousAIAnswer}} is accurate for this project source code: {{.SourceCode}}? Return a number from 1 to 10. Only return the number.`
	}
}
