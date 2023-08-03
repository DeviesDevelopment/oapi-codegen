package codegen

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
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
		code, err := Generate(swagger, opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, code.Output)
		assert.NotContains(t, code.Output[EchoServer].Code, `"/test/:name"`)
		assert.Contains(t, code.Output[EchoServer].Code, `"/cat"`)
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
		code, err := Generate(swagger, opts)
		assert.NoError(t, err)
		assert.NotEmpty(t, code.Output)
		assert.Contains(t, code.Output[EchoServer].Code, `"/test/:name"`)
		assert.NotContains(t, code.Output[EchoServer].Code, `"/cat"`)
	})
}
