package stub

import (
	"bytes"
	"strings"
	"testing"
)

func TestPythonAlgorithmTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
		ImportTypes: []string{"StructResult"},
		AllMetadata: []Metadata{
			{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
		},
		AllWindows: []Window{
			{
				VarName:     "FastWindow_1_0_0",
				Name:        "FastWindow",
				Version:     "1.0.0",
				Description: "A fast window type",
				Metadata: []Metadata{
					{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
				},
			},
		},
		Processors: []ProcessorData{
			{
				Name: "ml-test",
				Algorithms: []Algorithm{
					{
						Name:             "SpeedCheck",
						VarName:          "SpeedCheck_abc123",
						Version:          "1.1.0",
						WindowVarName:    "FastWindow_1_0_0",
						ReturnType:       structReturnType,
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
						Hash:             "abc123",
						Description:      "Checks speed of buses",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := pythonAlgoTemplate.Execute(&buf, testData)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	output := buf.String()

	assertions := []struct {
		name     string
		contains string
	}{
		{"Import ExecutionParams", "from orca_python import"},
		{"Import Result Type", "StructResult"},
		{"Function Definition", "def speed_check_abc123(params: ExecutionParams)"},
		{"Return Type Annotation", "-> StructResult:"},
		{"Metadata Comment", `# METADATA: {"Name": "SpeedCheck"`},
		{"Description in Docstring", "Checks speed of buses"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain expected string: %s", a.name, a.contains)
		}
	}

	t.Logf("Generated Python Algorithms:\n%s", output)
}

func TestPythonWindowTypeTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
		AllMetadata: []Metadata{
			{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
			{VarName: "route_id", KeyName: "route_id", Description: "Route identifier"},
		},
		AllWindows: []Window{
			{
				VarName:     "FastWindow_1_0_0",
				Name:        "FastWindow",
				Version:     "1.0.0",
				Description: "A fast window type",
				Metadata: []Metadata{
					{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
					{VarName: "route_id", KeyName: "route_id", Description: "Route identifier"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := pythonWindowTypeTemplate.Execute(&buf, testData)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	output := buf.String()

	assertions := []struct {
		name     string
		contains string
	}{
		{"Import WindowType", "from orca_python import WindowType"},
		{"Window Variable", "FastWindow_1_0_0: WindowType"},
		{"Window Description", "A fast window type"},
		{"Metadata Comment", `# METADATA: {"Name": "FastWindow", "Version": "1.0.0"`},
		{"Metadata Field in Docstring", "bus_id: Unique bus ID"},
		{"Metadata Field in Docstring 2", "route_id: Route identifier"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain expected string: %s", a.name, a.contains)
		}
	}

	t.Logf("Generated Python Window Types:\n%s", output)
}

func TestPythonMetadataTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
		AllMetadata: []Metadata{
			{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
			{VarName: "route_id", KeyName: "route_id", Description: "Route identifier"},
		},
	}

	var buf bytes.Buffer
	err := pythonMetadataTemplate.Execute(&buf, testData)
	if err != nil {
		t.Fatalf("Template execution failed: %v", err)
	}

	output := buf.String()

	assertions := []struct {
		name     string
		contains string
	}{
		{"Import MetadataField", "from orca_python import MetadataField"},
		{"Metadata Variable 1", "bus_id: MetadataField"},
		{"Metadata Variable 2", "route_id: MetadataField"},
		{"Description 1", "Unique bus ID"},
		{"Description 2", "Route identifier"},
		{"Metadata Comment 1", `# METADATA: {"Name": "bus_id"`},
		{"Metadata Comment 2", `# METADATA: {"Name": "route_id"`},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain expected string: %s", a.name, a.contains)
		}
	}

	t.Logf("Generated Python Metadata:\n%s", output)
}

func TestPythonTemplateGeneration_WithReturnTypes(t *testing.T) {
	testData := AllProcessors{
		ImportTypes: []string{"ValueResult", "StructResult", "NoneResult", "ArrayResult"},
		AllWindows: []Window{
			{
				VarName:     "Every30Second_1_0_0",
				Name:        "Every30Second",
				Version:     "1.0.0",
				Description: "30 second window",
				Metadata:    []Metadata{},
			},
		},
		Processors: []ProcessorData{
			{
				Name: "ml-test",
				Algorithms: []Algorithm{
					{
						Name:             "CalcAverage",
						VarName:          "CalcAverage_111",
						ReturnType:       valueReturnType,
						WindowVarName:    "Every30Second_1_0_0",
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
						Hash:             "111",
						Version:          "1.0.0",
					},
					{
						Name:             "GetBatch",
						VarName:          "GetBatch_222",
						ReturnType:       structReturnType,
						WindowVarName:    "Every30Second_1_0_0",
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
						Hash:             "222",
						Version:          "1.0.0",
					},
					{
						Name:             "SendResult",
						VarName:          "SendResult_333",
						ReturnType:       noneReturnType,
						WindowVarName:    "Every30Second_1_0_0",
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
						Hash:             "333",
						Version:          "1.0.0",
					},
					{
						Name:             "CalcDist",
						VarName:          "CalcDist_444",
						ReturnType:       arrayReturnType,
						WindowVarName:    "Every30Second_1_0_0",
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
						Hash:             "444",
						Version:          "1.0.0",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	pythonAlgoTemplate.Execute(&buf, testData)
	output := buf.String()

	assertions := []struct {
		name     string
		contains string
	}{
		{"ValueResult Import", "ValueResult"},
		{"StructResult Import", "StructResult"},
		{"ArrayResult Import", "ArrayResult"},
		{"NoneResult Import", "NoneResult"},
		{"ValueResult Function", "def calc_average_111(params: ExecutionParams)"},
		{"ValueResult Annotation", "-> ValueResult:"},
		{"StructResult Function", "def get_batch_222(params: ExecutionParams)"},
		{"StructResult Annotation", "-> StructResult:"},
		{"ArrayResult Function", "def calc_dist_444(params: ExecutionParams)"},
		{"ArrayResult Annotation", "-> ArrayResult:"},
		{"NoneResult Function", "def send_result_333(params: ExecutionParams)"},
		{"NoneResult Annotation", "-> NoneResult:"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Failed [%s]: Expected to find %s", a.name, a.contains)
		}
	}

	t.Logf("Generated Python:\n%s", output)
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SpeedCheck", "speed_check"},
		{"CalcAverage", "calc_average"},
		{"GetBatch", "get_batch"},
		{"SendResult", "send_result"},
		{"already_snake", "already_snake"},
		{"HTTPSConnection", "h_t_t_p_s_connection"},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitiseVariableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid_name", "valid_name"},
		{"name.with.dots", "name_with_dots"},
		{"9startsWithNumber", "_9startsWithNumber"},
		{"normal", "normal"},
	}

	for _, tt := range tests {
		result := sanitiseVariableName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitiseVariableName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
