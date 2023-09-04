package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// CodeOutput contains the generated code for one or more targets. The code is not
// formatted. To format the code, the client would call the WriteOutput() method,
// passing a boolean true to indicate that formatting is required.
type CodeOutput struct {
	Path string
	Name string
	Code string
}

// A map containing all the targets as key, and the generated code for each target.
// Multiple targets could potentially have been merged into the same output.
type GeneratedOutput map[string]*CodeOutput

// Returns the complete filename path of a target, including it's directories.
// Depending on the 'mdir' input argument, also creates the directories if
// needed.
func (f *CodeOutput) OutputPath(mkdir bool) string {
	s := strings.Split(f.Path, "/")
	p := filepath.Join(s...)

	if mkdir {
		os.MkdirAll(p, os.ModePerm)
	}
	return filepath.Join(p, f.Name)
}

// Outputs the generated code, either to a file, or to stdout if no filename has
// been given using the '-o' cmd flag.
func (g CodeOutput) WriteOutput() error {
	if g.Name != "" {
		if err := os.WriteFile(g.OutputPath(true), []byte(g.Code), 0o644); err != nil {
			return err
		}
		return nil
	}

	fmt.Print(g.Code)
	return nil
}

// Converts the generated code for all targets into a consolidated map of code output,
// where multiple targets sharing the same package and output options are merged into
// the same output. This means the generated map could contain multiple keys for the
// same output.
func GetGeneratedOutput(targets CodegenTargets, format bool) (GeneratedOutput, error) {
	var output GeneratedOutput = map[string]*CodeOutput{}
	// Used to keept track of the targets that have been merged
	merged := map[string]string{}

	// Loop through all targets
	for _, o := range targets {
		// Already merged?
		if merged[o.OutputPath(false)] != "" {
			continue
		}
		// Join imports and code
		s := strings.Join([]string{o.Imports, o.Code}, "\n")
		co := &CodeOutput{
			Path: o.Package,
			Name: o.FileName,
			Code: s,
		}
		output[o.Target] = co

		// Now loop through targets again to see if code needs to be merged
		// with other targets
		for _, t := range targets {
			// Don't compare with self
			if o.Target == t.Target {
				continue
			}

			// Do they share package and output file name?
			if o.OutputPath(false) == t.OutputPath(false) {
				// Merge code into existing file
				s := strings.Join([]string{co.Code, t.Code}, "\n")
				co.Code = s
				output[t.Target] = co
				merged[o.OutputPath(false)] = "true"
			}
		}
	}
	// Now, format the code if needed
	if format {
		for _, o := range output {
			formattedCode, err := imports.Process(o.Name, []byte(o.Code), nil)
			if err != nil {
				return nil, err
			}
			o.Code = string(formattedCode)
		}
	}
	return output, nil
}

// Convenience method that flattens a map of generated CodeOutput into an array,
// making it easier to iterate over it to print the output to file or stdout.
func (g GeneratedOutput) ToArray() []*CodeOutput {
	result := []*CodeOutput{}

	// Loop through the map
	for _, c := range g {
		found := false
		// Check if result contains the output already
		for _, o := range result {
			if c.OutputPath(false) == o.OutputPath(false) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, c)
		}
	}
	return result
}
