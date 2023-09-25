package grabimportnames

import (
	"testing"

	"github.com/DeviesDevelopment/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestLineComments(t *testing.T) {
	swagger, err := openapi3.NewLoader().LoadFromFile("spec.yaml")
	require.NoError(t, err)

	opts := codegen.NewDefaultConfigurationWithPackage("grabimportnames")
	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	require.NoError(t, err)

	target, _ := output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)
	require.NotContains(t, target.Code, `"openapi_types"`)
}
