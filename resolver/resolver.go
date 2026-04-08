package resolver

import (
	"fmt"
	"sort"

	"github.com/woncomp/grapes/parser"
)

// Resolve takes the import list from the master file and all known fragments,
// and returns them in topological order based on deps.
// Fragments not reachable from imports are excluded.
// Returns an error if there are cycles or missing dependencies.
func Resolve(imports []string, fragments []*parser.Fragment) ([]*parser.Fragment, error) {
	fragMap := make(map[string]*parser.Fragment, len(fragments))
	for _, f := range fragments {
		fragMap[f.Name] = f
	}

	// Collect all reachable fragments (imports + transitive deps)
	visited := make(map[string]bool)
	var collect func(name string) error
	collect = func(name string) error {
		if visited[name] {
			return nil
		}
		f, ok := fragMap[name]
		if !ok {
			return fmt.Errorf("missing fragment: %s", name)
		}
		visited[name] = true
		for _, dep := range f.Deps {
			if err := collect(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for _, name := range imports {
		if err := collect(name); err != nil {
			return nil, err
		}
	}

	// Build adjacency list for reachable fragments
	inDegree := make(map[string]int)
	edges := make(map[string][]string)

	for name := range visited {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		f := fragMap[name]
		for _, dep := range f.Deps {
			edges[dep] = append(edges[dep], name)
			inDegree[name]++
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // stable order for no-dep fragments

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		next := edges[current]
		sort.Strings(next) // deterministic
		for _, n := range next {
			inDegree[n]--
			if inDegree[n] == 0 {
				queue = append(queue, n)
				sort.Strings(queue)
			}
		}
	}

	if len(sorted) != len(visited) {
		// Find the cycle
		cycle := findCycle(visited, fragMap)
		return nil, fmt.Errorf("circular dependency: %s", cycle)
	}

	result := make([]*parser.Fragment, len(sorted))
	for i, name := range sorted {
		result[i] = fragMap[name]
	}

	return result, nil
}

// findCycle returns a human-readable cycle description.
func findCycle(visited map[string]bool, fragMap map[string]*parser.Fragment) string {
	for start := range visited {
		path := []string{start}
		seen := map[string]bool{start: true}
		if dfs(start, path, seen, fragMap, &path) {
			last := path[len(path)-1]
			cycleStart := -1
			for i, p := range path[:len(path)-1] {
				if p == last {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := path[cycleStart:]
				cycle = append(cycle, last)
				result := ""
				for i, p := range cycle {
					if i > 0 {
						result += " -> "
					}
					result += p
				}
				return result
			}
		}
	}
	return "unknown cycle"
}

func dfs(current string, path []string, seen map[string]bool, fragMap map[string]*parser.Fragment, result *[]string) bool {
	f := fragMap[current]
	for _, dep := range f.Deps {
		if seen[dep] {
			*result = append(path, dep)
			return true
		}
		seen[dep] = true
		newPath := append(path, dep)
		if dfs(dep, newPath, seen, fragMap, result) {
			return true
		}
		delete(seen, dep)
	}
	return false
}
