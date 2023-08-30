package grabimportnames

import (
	"fmt"
	"testing"

	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestLineComments(t *testing.T) {
	swagger, err := openapi3.NewLoader().LoadFromFile("spec.yaml")
	require.NoError(t, err)

	opts := codegen.NewDefaultConfigurationWithPackage("grabimportnames")
	err = codegen.Generate(swagger, opts)

	require.NoError(t, err)
	target := opts.GetTarget(codegen.Models)
	if target == nil {
		fmt.Println("Model target not found")
	}
	code := opts.GetTarget(codegen.Models).GetOutput(true)
	require.NotContains(t, code, `"openapi_types"`)
}
