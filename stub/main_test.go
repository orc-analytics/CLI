package stub

import (
	"bytes"
	"strings"
	"testing"
)

func TestPythonAlgorithmTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
		ImportTypes: []string{"StructResult"},
		Processors: []ProcessorData{
			{
				Name: "ml-test",
				Algorithms: []Algorithm{
					{
						Name:             "SpeedCheck",
						VarName:          "SpeedCheck_abc123",
						Version:          "1.1.0",
						ReturnType:       "StructResult",
						ProcessorName:    "ml-test",
						ProcessorRuntime: "python",
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
		{"Export List", "__all__ = ["},
		{"Export Item", `"speed_check_abc123"`},
		{"Function Definition", "def speed_check_abc123(params: ExecutionParams) -> StructResult:"},
		{"NotImplementedError", "raise NotImplementedError"},
		{"Remote Attribute", "speed_check_abc123.__orca_is_remote__ = True  # type: ignore"},
		{"Metadata Attribute", "speed_check_abc123.__orca_metadata__ = {"},
		{"Metadata Content", `"Name": "SpeedCheck"`},
		{"Description in Docstring", "Checks speed of buses"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain: %s", a.name, a.contains)
		}
	}
}

func TestPythonMetadataTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
		AllMetadata: []Metadata{
			{VarName: "bus_id", KeyName: "bus_id", Description: "Unique bus ID"},
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
		{"Internal Class", "class _Field:"},
		{"Export List", "__all__ = ["},
		{"Variable Assignment", "bus_id: MetadataField = _Field("},
		{"Type Ignore", "# type: ignore"},
		{"Metadata Dict", `"Name": "bus_id"`},
		{"Docstring", `"""Unique bus ID"""`},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain: %s", a.name, a.contains)
		}
	}
}

func TestPythonWindowTypeTemplateGeneration(t *testing.T) {
	testData := AllProcessors{
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
		{"Internal Window Class", "class _Window:"},
		{"Variable Assignment", "FastWindow_1_0_0: WindowType = _Window("},
		{"Metadata Field Instantiation", `_Field(name="bus_id", description="Unique bus ID")`},
		{"Window Metadata", `"Name": "FastWindow"`},
		{"Nested Metadata", `"MetadataFields": [`},
		{"Docstring Description", "A fast window type"},
		{"Docstring Field", "- bus_id: Unique bus ID"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Assertion Failed [%s]: Output did not contain: %s", a.name, a.contains)
		}
	}
}

func TestPythonTemplateGeneration_WithReturnTypes(t *testing.T) {
	testData := AllProcessors{
		ImportTypes: []string{"ValueResult", "ArrayResult"},
		Processors: []ProcessorData{
			{
				Algorithms: []Algorithm{
					{
						VarName:    "CalcAverage_111",
						ReturnType: "ValueResult",
					},
					{
						VarName:    "CalcDist_444",
						ReturnType: "ArrayResult",
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
		{"ValueResult Annotation", "-> ValueResult:"},
		{"ArrayResult Annotation", "-> ArrayResult:"},
		{"Attribute Assignment 1", "calc_average_111.__orca_is_remote__"},
		{"Attribute Assignment 2", "calc_dist_444.__orca_is_remote__"},
	}

	for _, a := range assertions {
		if !strings.Contains(output, a.contains) {
			t.Errorf("Failed [%s]: Expected %s", a.name, a.contains)
		}
	}
}

// ... helper tests (ToSnakeCase, SanitiseVariableName) remain unchanged ...

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
