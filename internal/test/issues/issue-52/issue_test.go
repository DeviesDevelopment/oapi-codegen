package issue52

import (
	_ "embed"
	"testing"

	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

//go:embed spec.yaml
var spec []byte

func TestIssue(t *testing.T) {
	swagger, err := openapi3.NewLoader().LoadFromData(spec)
	require.NoError(t, err)

	opts := codegen.NewDefaultConfigurationWithPackage("issue52")

	err = codegen.Generate(swagger, opts)
	require.NoError(t, err)
}
