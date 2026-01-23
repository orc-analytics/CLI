package stub

import (
	"embed"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	pb "github.com/orca-telemetry/core/protobufs/go"
)

const (
	PYTHON_METADATA_FIELDS_TMPL = "stub_templates/window_metadata_fields.py.tmpl"
	PYTHON_WINDOW_TYPES_TMPL    = "stub_templates/window_types.py.tmpl"
	PYTHON_ALGORITHMS_TMPL      = "stub_templates/algorithms.py.tmpl"
)

//go:embed stub_templates/*.tmpl
var templateFS embed.FS

var (
	pythonAlgoTemplate       *template.Template
	pythonMetadataTemplate   *template.Template
	pythonWindowTypeTemplate *template.Template
)

type ReturnType string

const (
	structReturnType ReturnType = "StructResult"
	valueReturnType  ReturnType = "ValueResult"
	noneReturnType   ReturnType = "NoneResult"
	arrayReturnType  ReturnType = "ArrayResult"
)

func generateTemplate(templatePath string) *template.Template {
	baseName := filepath.Base(templatePath)
	parsedTemplate := template.Must(template.New(baseName).Funcs(
		template.FuncMap{
			"ToSnakeCase":          toSnakeCase,
			"SanitiseVariableName": sanitiseVariableName,
			"WrapText":             wrapText,
			"Indent":               pythonIndent,
		}).ParseFS(templateFS, templatePath))
	return parsedTemplate
}
func init() {
	pythonAlgoTemplate = generateTemplate(PYTHON_ALGORITHMS_TMPL)
	pythonMetadataTemplate = generateTemplate(PYTHON_METADATA_FIELDS_TMPL)
	pythonWindowTypeTemplate = generateTemplate(PYTHON_WINDOW_TYPES_TMPL)
}

func wrapText(limit int, text string) string {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	currentLineLength := 0

	for i, word := range words {
		// If adding the word + a space exceeds the limit
		if currentLineLength+len(word) > limit && currentLineLength > 0 {
			result.WriteString("\n")
			currentLineLength = 0
		} else if i > 0 {
			result.WriteString(" ")
			currentLineLength++
		}

		result.WriteString(word)
		currentLineLength += len(word)
	}

	return result.String()
}

// Helper to indent lines for Python docstrings
func pythonIndent(spaces int, text string) string {
	prefix := strings.Repeat(" ", spaces)
	return prefix + strings.ReplaceAll(text, "\n", "\n"+prefix)
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func sanitiseVariableName(s string) string {
	var result []rune
	for i, r := range s {
		if i == 0 {
			if _, err := strconv.Atoi(string(r)); err == nil {
				result = append(result, '_')
				result = append(result, r)
				continue
			}
		}
		if r == '.' {
			result = append(result, '_')
		} else {
			result = append(result, r)
		}

	}
	return string(result)
}

// data structures matching the template expectations
type Metadata struct {
	VarName     string
	KeyName     string
	Description string
}

type Window struct {
	VarName     string
	Name        string
	Version     string
	Description string
	Metadata    []Metadata
}

type Algorithm struct {
	Name             string
	VarName          string
	Description      string
	ProcessorName    string
	ProcessorRuntime string
	Version          string
	WindowVarName    string
	ReturnType       ReturnType
	Hash             string
}

type ProcessorData struct {
	Name       string
	Metadata   []Metadata
	Windows    []Window
	Algorithms []Algorithm
}

type AllProcessors struct {
	Processors  []ProcessorData
	ImportTypes []string
	AllMetadata []Metadata
	AllWindows  []Window
}

func mapInternalStateToTmpl(internalState *pb.InternalState) (error, *AllProcessors) {
	processorDatas := make([]ProcessorData, len(internalState.GetProcessors()))

	usedReturnTypes := make(map[string]bool)
	globalMetadataMap := make(map[string]Metadata)
	globalWindowsMap := make(map[string]Window)

	for ii, proc := range internalState.GetProcessors() {
		supportedAlgorithms := make([]Algorithm, len(proc.GetSupportedAlgorithms()))

		for jj, algo := range proc.GetSupportedAlgorithms() {
			windowName := algo.GetWindowType().GetName()
			windowVer := algo.GetWindowType().GetVersion()
			windowKey := fmt.Sprintf("%v_%v", windowName, windowVer)

			metadataForWindow := make([]Metadata, len(algo.GetWindowType().GetMetadataFields()))
			for kk, metadata := range algo.GetWindowType().GetMetadataFields() {
				mName := metadata.GetName()

				if _, ok := globalMetadataMap[mName]; !ok {
					globalMetadataMap[mName] = Metadata{
						VarName:     mName,
						KeyName:     mName,
						Description: metadata.GetDescription(),
					}
				}
				metadataForWindow[kk] = Metadata{
					VarName:     mName,
					KeyName:     mName,
					Description: metadata.GetDescription(),
				}
			}

			if _, ok := globalWindowsMap[windowKey]; !ok {
				globalWindowsMap[windowKey] = Window{
					VarName:     windowKey,
					Name:        windowName,
					Version:     windowVer,
					Description: algo.GetWindowType().GetDescription(),
					Metadata:    metadataForWindow,
				}
			}

			var algoReturnType ReturnType
			switch algo.GetResultType() {
			case pb.ResultType_ARRAY:
				algoReturnType = arrayReturnType
			case pb.ResultType_STRUCT:
				algoReturnType = structReturnType
			case pb.ResultType_VALUE:
				algoReturnType = valueReturnType
			case pb.ResultType_NONE:
				algoReturnType = noneReturnType
			case pb.ResultType_NOT_SPECIFIED:
				return fmt.Errorf(
					"result type not specified for algorithm %v_%v on processor %v_%v",
					algo.GetName(),
					algo.GetVersion(),
					proc.GetName(),
					proc.GetRuntime(),
				), nil
			}
			usedReturnTypes[string(algoReturnType)] = true

			h := crc32.NewIEEE()
			h.Write([]byte(proc.GetName()))
			h.Write([]byte(proc.GetConnectionStr()))
			h.Write([]byte(windowName))
			h.Write([]byte(windowVer))
			h.Write([]byte(algo.GetName()))
			h.Write([]byte(algo.GetVersion()))
			algorithmHash := h.Sum32()

			supportedAlgorithms[jj] = Algorithm{
				Name:             algo.GetName(),
				VarName:          fmt.Sprintf("%v_%x", algo.GetName(), algorithmHash),
				ProcessorName:    proc.GetName(),
				ProcessorRuntime: proc.GetRuntime(),
				Version:          algo.GetVersion(),
				ReturnType:       algoReturnType,
				WindowVarName:    windowKey,
				Hash:             fmt.Sprintf("%x", algorithmHash),
				Description:      algo.GetDescription(),
			}
		}

		processorDatas[ii] = ProcessorData{
			Name:       proc.GetName(),
			Algorithms: supportedAlgorithms,
		}
	}

	// Convert Global Metadata Map to Slice
	allMetadata := make([]Metadata, 0, len(globalMetadataMap))
	for _, m := range globalMetadataMap {
		allMetadata = append(allMetadata, m)
	}

	// Convert Global Windows Map to Slice
	allWindows := make([]Window, 0, len(globalWindowsMap))
	for _, w := range globalWindowsMap {
		allWindows = append(allWindows, w)
	}

	// Finalize Import List
	importList := []string{}
	availableTypes := []string{"StructResult", "ValueResult", "NoneResult", "ArrayResult"}
	for _, t := range availableTypes {
		if usedReturnTypes[t] {
			importList = append(importList, t)
		}
	}

	return nil, &AllProcessors{
		Processors:  processorDatas,
		ImportTypes: importList,
		AllMetadata: allMetadata,
		AllWindows:  allWindows,
	}
}

func GeneratePythonStubs(internalState *pb.InternalState, outDir string) error {

	err, tmplData := mapInternalStateToTmpl(internalState)
	if err != nil {
		return fmt.Errorf("could not parse internal state: %w", err)
	}

	err = os.Mkdir(outDir, 0750)
	err = os.MkdirAll(filepath.Join(outDir, "registry"), 0750)

	if err != nil && !os.IsExist(err) {
		return (err)
	}

	initFile, err := os.Create(filepath.Join(outDir, "registry", "__init__.py"))

	if err != nil && !os.IsExist(err) {
		return err
	}
	initFile.Close()

	algorithmsFile, err := os.Create(filepath.Join(outDir, "registry", "algorithms.py"))
	if err != nil && !os.IsExist(err) {
		return err
	}
	defer algorithmsFile.Close()

	windowTypesFile, err := os.Create(filepath.Join(outDir, "registry", "window_types.py"))
	if err != nil && !os.IsExist(err) {
		return err
	}
	defer windowTypesFile.Close()

	metadataFieldsFile, err := os.Create(filepath.Join(outDir, "registry", "metadata_fields.py"))
	if err != nil && !os.IsExist(err) {
		return err
	}
	defer metadataFieldsFile.Close()

	if err := pythonAlgoTemplate.Execute(algorithmsFile, tmplData); err != nil {
		panic(err)
	}
	if err := pythonWindowTypeTemplate.Execute(windowTypesFile, tmplData); err != nil {
		panic(err)
	}
	if err := pythonMetadataTemplate.Execute(metadataFieldsFile, tmplData); err != nil {
		panic(err)
	}
	return nil
}
