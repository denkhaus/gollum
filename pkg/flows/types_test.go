package flows

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowStruct_BasicFields(t *testing.T) {
	flow := &Flow{
		Name:    "test-flow",
		Version: "1.0",
	}

	assert.Equal(t, "test-flow", flow.Name)
	assert.Equal(t, "1.0", flow.Version)
}

func TestInputBlock_HasRequiredAndType(t *testing.T) {
	input := &InputBlock{
		Ints: []FieldDef{
			{Name: "pr_number", Required: true},
		},
		Strings: []FieldDef{
			{Name: "repo_owner", Default: "denkhaus"},
		},
	}

	fields := input.GetAllFields()
	assert.Len(t, fields, 2)

	// Strings come first, then Ints (per implementation order)
	assert.Equal(t, "repo_owner", fields[0].Name)
	assert.Equal(t, "denkhaus", fields[0].Default)

	assert.Equal(t, "pr_number", fields[1].Name)
	assert.True(t, fields[1].Required)
}

func TestComputedBlock_HasComputedFields(t *testing.T) {
	computed := &ComputedBlock{
		Bools: []ComputedFieldDef{
			{Name: "is_open", Type: "bool", Eval: "EQ(context.status, 'open')"},
			{Name: "is_large", Type: "bool", Eval: "GT(context.count, 10)"},
		},
	}

	assert.Len(t, computed.GetAllFields(), 2)
	assert.Equal(t, "is_open", computed.GetAllFields()[0].Name)
	assert.Equal(t, TypeBool, computed.GetAllFields()[0].Type)
	assert.Equal(t, "EQ(context.status, 'open')", computed.GetAllFields()[0].Eval)
}

func TestFlow_HasComputedBlock(t *testing.T) {
	flow := &Flow{
		Name: "test-flow",
		Computed: &ComputedBlock{
			Strings: []ComputedFieldDef{
				{Name: "result", Type: "string", Eval: "CONCAT(input.prefix, input.suffix)"},
			},
		},
	}

	assert.NotNil(t, flow.Computed)
	assert.Len(t, flow.Computed.GetAllFields(), 1)
	assert.Equal(t, "result", flow.Computed.GetAllFields()[0].Name)
}

func TestFieldDef_AssignFromAttribute(t *testing.T) {
	xmlData := `<string name="result" assignFrom="computed.sum" />`
	var field FieldDef
	err := xml.Unmarshal([]byte(xmlData), &field)
	assert.NoError(t, err)
	assert.Equal(t, "result", field.Name)
	assert.Equal(t, "computed.sum", field.AssignFrom)
}

func TestOutputBlock_GetDeclarative(t *testing.T) {
	block := &OutputBlock{
		Ints: []FieldDef{
			{Name: "sum", AssignFrom: "computed.sum", Type: TypeInt},    // declarative
			{Name: "count", Type: TypeInt},                        // imperative
		},
	}

	declarative := block.GetDeclarative()
	assert.Len(t, declarative, 1)
	assert.Equal(t, "sum", declarative[0].Name)

	imperative := block.GetImperative()
	assert.Len(t, imperative, 1)
	assert.Equal(t, "count", imperative[0].Name)
}

func TestStep_VerboseAttribute(t *testing.T) {
	xmlData := `<step type="llm" agent="test" verbose="true">
		<prompt>Test</prompt>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xmlData), &step)
	assert.NoError(t, err)
	assert.True(t, step.Verbose, "verbose should be true when set to true")
}

func TestStep_VerboseAttributeDefault(t *testing.T) {
	xmlData := `<step type="llm" agent="test">
		<prompt>Test</prompt>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xmlData), &step)
	assert.NoError(t, err)
	assert.False(t, step.Verbose, "verbose should default to false when omitted")
}

func TestStep_StepResultXMLParsing(t *testing.T) {
	// Test that <result> with assignTo attribute parses correctly
	xmlData := `<step type="shell" name="test">
		<cmd>echo hello</cmd>
		<result assignTo="output.stdout">
			<string path="stdout" assignTo="stdout"/>
		</result>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xmlData), &step)
	assert.NoError(t, err)
	assert.NotNil(t, step.Result, "Result should be parsed")
	assert.Equal(t, "output.stdout", step.Result.AssignTo, "assignTo attribute should be parsed")
	assert.Len(t, step.Result.Paths, 1, "Should have one result path")

	path := step.Result.Paths[0]
	assert.Equal(t, "stdout", path.Path, "path attribute should be parsed")
	assert.Equal(t, "stdout", path.AssignTo, "assignTo attribute should be parsed")
}

func TestStep_StepResultSimpleAssign(t *testing.T) {
	// Test simple assign without nested paths
	xmlData := `<step type="llm" agent="worker">
		<prompt>Analyze this</prompt>
		<result assignTo="output.analysis"/>
	</step>`

	var step Step
	err := xml.Unmarshal([]byte(xmlData), &step)
	assert.NoError(t, err)
	assert.NotNil(t, step.Result, "Result should be parsed")
	assert.Equal(t, "output.analysis", step.Result.AssignTo, "assignTo should be parsed")
	assert.Empty(t, step.Result.Paths, "Paths should be empty for simple assign")
}

func TestAgentStrategyParsing(t *testing.T) {
	xmlData := `<agents>
		<agent name="test">
			<strategy maxIterations="10" maxRepeatedActions="2" type="react"/>
			<prompt>Test prompt</prompt>
		</agent>
		<agent name="no-strategy">
			<prompt>No strategy</prompt>
		</agent>
	</agents>`

	var structWithAgents struct {
		Agents []Agent `xml:"agent"`
	}
	err := xml.Unmarshal([]byte(xmlData), &structWithAgents)
	require.NoError(t, err)

	agents := structWithAgents.Agents
	require.Len(t, agents, 2)

	// Agent with strategy
	assert.NotNil(t, agents[0].Strategy)
	assert.Equal(t, 10, agents[0].Strategy.MaxIterations)
	assert.Equal(t, 2, agents[0].Strategy.MaxRepeatedActions)
	assert.Equal(t, "react", string(agents[0].Strategy.Type))

	// Agent without strategy
	assert.Nil(t, agents[1].Strategy)
}

func TestFlowVariableScopeEnv(t *testing.T) {
	scope := FlowVariableScopeEnv
	if scope != "env" {
		t.Errorf("expected 'env', got %q", scope)
	}

	err := scope.Validate()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
