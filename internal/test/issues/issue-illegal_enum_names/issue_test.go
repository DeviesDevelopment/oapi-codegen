package illegalenumnames

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/DeviesDevelopment/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestIllegalEnumNames(t *testing.T) {
	swagger, err := openapi3.NewLoader().LoadFromFile("spec.yaml")
	require.NoError(t, err)

	opts := codegen.NewDefaultConfigurationWithPackage("illegalenumnames")

	output, err := codegen.Generate(swagger, opts)
	require.NotNil(t, output, "No targets found")
	require.NoError(t, err)

	target, _ := output[codegen.Models]
	require.NotNil(t, target, "Expected target not found: %s", codegen.Models)

	f, err := parser.ParseFile(token.NewFileSet(), "", target.Code, parser.AllErrors)
	require.NoError(t, err)

	constDefs := make(map[string]string)
	for _, d := range f.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			if token.CONST == decl.Tok {
				for _, s := range decl.Specs {
					switch spec := s.(type) {
					case *ast.ValueSpec:
						constDefs[spec.Names[0].Name] = spec.Names[0].Obj.Decl.(*ast.ValueSpec).Values[0].(*ast.BasicLit).Value
					}
				}
			}
		}
	}

	require.Equal(t, `""`, constDefs["BarEmpty"])
	require.Equal(t, `"Bar"`, constDefs["BarBar"])
	require.Equal(t, `"Foo"`, constDefs["BarFoo"])
	require.Equal(t, `"Foo Bar"`, constDefs["BarFooBar"])
	require.Equal(t, `"Foo-Bar"`, constDefs["BarFooBar1"])
	require.Equal(t, `"1Foo"`, constDefs["BarN1Foo"])
	require.Equal(t, `" Foo"`, constDefs["BarFoo1"])
	require.Equal(t, `" Foo "`, constDefs["BarFoo2"])
	require.Equal(t, `"_Foo_"`, constDefs["BarFoo3"])
	require.Equal(t, `"1"`, constDefs["BarN1"])
}
