package codegen

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterOperationsByTag(t *testing.T) {
	t.Run("include tags", func(t *testing.T) {
		opts := NewDefaultConfigurationWithPackage("testswagger")
		opts.OutputOptions = OutputOptions{
			IncludeTags: []string{"hippo", "giraffe", "cat"},
		}

		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true

		// Get a spec from the test definition in this file:
		swagger, err := loader.LoadFromData([]byte(testOpenAPIDefinition))
		assert.NoError(t, err)

		// Run our code generation:
		_, err = Generate(swagger, opts)
		assert.NoError(t, err)

		target, exists := opts.Targets[EchoServer]
		assert.Equal(t, exists, true)

		code := target.GetOutput(true)
		assert.NotEmpty(t, code)
		assert.NotContains(t, code, `"/test/:name"`)
		assert.Contains(t, code, `"/cat"`)
	})

	t.Run("exclude tags", func(t *testing.T) {
		opts := NewDefaultConfigurationWithPackage("testswagger")
		opts.OutputOptions = OutputOptions{
			ExcludeTags: []string{"hippo", "giraffe", "cat"},
		}

		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true

		// Get a spec from the test definition in this file:
		swagger, err := loader.LoadFromData([]byte(testOpenAPIDefinition))
		assert.NoError(t, err)

		// Run our code generation:
		output, err := Generate(swagger, opts)
		require.NotNil(t, output, "No targets found")
		assert.NoError(t, err)

		target, _ := output[EchoServer]
		require.NotNil(t, target, "Expected target not found: %s", EchoServer)

		assert.NotEmpty(t, target.Code)
		assert.Contains(t, target.Code, `"/test/:name"`)
		assert.NotContains(t, target.Code, `"/cat"`)
	})
}
