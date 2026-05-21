package resolver

import (
	"fmt"

	"github.com/woncomp/grapes/parser"
)

// Resolve takes the ordered import list from the master TOML file and all known grapes,
// validates that each imported grape exists, and returns them in import order.
func Resolve(imports []parser.GrapeImport, grapes []*parser.GrapeFile) ([]*parser.GrapeFile, error) {
	grapeMap := make(map[string]*parser.GrapeFile, len(grapes))
	for _, grape := range grapes {
		grapeMap[grapeIdentityKey(grape)] = grape
	}

	resolved := make([]*parser.GrapeFile, 0, len(imports))
	seen := make(map[string]bool, len(imports))
	for _, imp := range imports {
		if seen[imp.Key] {
			continue
		}
		grape, ok := grapeMap[imp.Key]
		if !ok {
			return nil, fmt.Errorf("missing fragment: %s", imp.Label)
		}
		seen[imp.Key] = true
		resolved = append(resolved, grape)
	}
	return resolved, nil
}

func grapeIdentityKey(grape *parser.GrapeFile) string {
	if grape == nil {
		return ""
	}
	if grape.Key != "" {
		return grape.Key
	}
	return grape.Name
}
