package issue1093

import (
	_ "embed"
	"testing"

	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

//go:embed child.api.yaml
var spec []byte

func TestIssue(t *testing.T) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	swagger, err := loader.LoadFromData(spec)
	require.NoError(t, err)

	opts := codegen.NewDefaultConfigurationWithPackage("issue1093")
	opts.ImportMapping = map[string]string{
		"parent.api.yaml": "github.com/deepmap/oapi-codegen/internal/test/issues/issue-1093/api/parent",
	}

	err = codegen.Generate(swagger, opts)
	require.NoError(t, err)
}
