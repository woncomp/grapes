package resolver

import (
	"fmt"

	"github.com/woncomp/grapes/parser"
)

// Resolve takes the import list from the .grapes file and all known grapes,
// validates that each imported grape exists, and returns them in import order.
func Resolve(imports []string, grapes []*parser.GrapeFile) ([]*parser.GrapeFile, error) {
	grapeMap := make(map[string]*parser.GrapeFile, len(grapes))
	for _, grape := range grapes {
		grapeMap[grape.Name] = grape
	}

	resolved := make([]*parser.GrapeFile, 0, len(imports))
	seen := make(map[string]bool, len(imports))
	for _, name := range imports {
		if seen[name] {
			continue
		}
		grape, ok := grapeMap[name]
		if !ok {
			return nil, fmt.Errorf("missing fragment: %s", name)
		}
		seen[name] = true
		resolved = append(resolved, grape)
	}
	return resolved, nil
}
