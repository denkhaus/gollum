package executor

import (
	"fmt"
	"testing"

	"github.com/denkhaus/gollum/pkg/flows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/samber/do/v2"
)

// TestRuntimeInputValidation tests that input values are validated against their type definitions
func TestRuntimeInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		flow        *flows.Flow
		inputs      map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid string input",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"name": "Alice"},
			wantErr: false,
		},
		{
			name: "valid int input",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Ints: []flows.FieldDef{{Name: "age", Type: flows.TypeInt}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"age": "25"},
			wantErr: false,
		},
		{
			name: "valid bool input true",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Bools: []flows.FieldDef{{Name: "active", Type: flows.TypeBool}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"active": "true"},
			wantErr: false,
		},
		{
			name: "valid bool input false",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Bools: []flows.FieldDef{{Name: "active", Type: flows.TypeBool}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"active": "false"},
			wantErr: false,
		},
		{
			name: "valid float input",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Floats: []flows.FieldDef{{Name: "price", Type: flows.TypeFloat}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"price": "19.99"},
			wantErr: false,
		},
		{
			name: "invalid int input - not a number",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Ints: []flows.FieldDef{{Name: "age", Type: flows.TypeInt}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:      map[string]string{"age": "twenty-five"},
			wantErr:     true,
			errContains: "invalid input for 'age'",
		},
		{
			name: "invalid int input - negative zero",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Ints: []flows.FieldDef{{Name: "age", Type: flows.TypeInt}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{"age": "-1"},
			wantErr: false, // negative is valid for int
		},
		{
			name: "invalid bool input - not a boolean",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Bools: []flows.FieldDef{{Name: "active", Type: flows.TypeBool}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:      map[string]string{"active": "yes"},
			wantErr:     true,
			errContains: "invalid input for 'active'",
		},
		{
			name: "invalid float input - not a number",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Floats: []flows.FieldDef{{Name: "price", Type: flows.TypeFloat}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:      map[string]string{"price": "free"},
			wantErr:     true,
			errContains: "invalid input for 'price'",
		},
		{
			name: "missing required field with no default",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString, Required: true}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:      map[string]string{},
			wantErr:     true,
			errContains: "missing required input",
		},
		{
			name: "missing optional field - should not error",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString, Required: false}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{},
			wantErr: false,
		},
		{
			name: "field with default value - should use default",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString, Default: "Anonymous"}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:  map[string]string{},
			wantErr: false,
		},
		{
			name: "unknown field - should error",
			flow: &flows.Flow{
				Name: "test-flow",
				Input: &flows.InputBlock{
					Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString}},
				},
				States: []flows.State{{Name: "init", Initial: true}},
			},
			inputs:      map[string]string{"unknown": "value"},
			wantErr:     true,
			errContains: "unknown input field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := setupTestDI(t)
			service := do.MustInvoke[FlowExecutorService](injector)
			instance := service.New(tt.flow)

			err := instance.SetInput(tt.inputs)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInputValidationWithMultipleFields tests validation with multiple fields of different types
func TestInputValidationWithMultipleFields(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{{Name: "name", Type: flows.TypeString, Required: true}},
			Ints:    []flows.FieldDef{{Name: "age", Type: flows.TypeInt}},
			Bools:   []flows.FieldDef{{Name: "active", Type: flows.TypeBool}},
			Floats:  []flows.FieldDef{{Name: "score", Type: flows.TypeFloat}},
		},
		States: []flows.State{{Name: "init", Initial: true}},
	}

	t.Run("all valid inputs", func(t *testing.T) {
		injector := setupTestDI(t)
		service := do.MustInvoke[FlowExecutorService](injector)
		instance := service.New(flow)

		inputs := map[string]string{
			"name":   "Alice",
			"age":    "30",
			"active": "true",
			"score":  "95.5",
		}

		err := instance.SetInput(inputs)
		require.NoError(t, err)
	})

	t.Run("one invalid input fails all", func(t *testing.T) {
		injector := setupTestDI(t)
		service := do.MustInvoke[FlowExecutorService](injector)
		instance := service.New(flow)

		inputs := map[string]string{
			"name":   "Alice",
			"age":    "thirty", // invalid
			"active": "true",
			"score":  "95.5",
		}

		err := instance.SetInput(inputs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input for 'age'")
	})
}

// TestInputValidationWithDefaults tests that default values are used when inputs are missing
func TestInputValidationWithDefaults(t *testing.T) {
	flow := &flows.Flow{
		Name: "test-flow",
		Input: &flows.InputBlock{
			Strings: []flows.FieldDef{
				{Name: "name", Type: flows.TypeString, Default: "Anonymous"},
				{Name: "greeting", Type: flows.TypeString, Default: "Hello"},
			},
			Ints: []flows.FieldDef{
				{Name: "count", Type: flows.TypeInt, Default: "10"},
			},
		},
		States: []flows.State{{Name: "init", Initial: true}},
	}

	t.Run("use defaults for missing fields", func(t *testing.T) {
		injector := setupTestDI(t)
		service := do.MustInvoke[FlowExecutorService](injector)
		instance := service.New(flow)

		inputs := map[string]string{
			"name": "Alice", // override default
			// greeting uses default
			// count uses default
		}

		err := instance.SetInput(inputs)
		require.NoError(t, err)

		// Verify defaults were set - use GetInput for input fields
		ctx := instance.GetContext()
		nameVal := ctx.GetInput("name")
		assert.Equal(t, "Alice", nameVal)

		// For greeting and count, we need to check the context values
		// Since they have defaults, they should be accessible
		greetingVal := ctx.GetInput("greeting")
		assert.Equal(t, "Hello", greetingVal)

		countVal := ctx.GetInput("count")
		// count is an int, so convert to string for comparison
		assert.Equal(t, "10", fmt.Sprintf("%v", countVal))
	})
}
