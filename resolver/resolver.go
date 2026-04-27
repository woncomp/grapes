package resolver

import (
	"fmt"
	"sort"

	"github.com/woncomp/grapes/parser"
)

// Resolve takes the import list from the .grapes file and all known grapes,
// and returns them in topological order based on deps.
// Grapes not reachable from imports are excluded.
// Returns an error if there are cycles or missing dependencies.
func Resolve(imports []string, grapes []*parser.GrapeFile) ([]*parser.GrapeFile, error) {
	grapeMap := make(map[string]*parser.GrapeFile, len(grapes))
	for _, grape := range grapes {
		grapeMap[grape.Name] = grape
	}

	visited := make(map[string]bool)
	var collect func(name string) error
	collect = func(name string) error {
		if visited[name] {
			return nil
		}
		grape, ok := grapeMap[name]
		if !ok {
			return fmt.Errorf("missing fragment: %s", name)
		}
		visited[name] = true
		for _, dep := range grape.Deps {
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

	inDegree := make(map[string]int)
	edges := make(map[string][]string)
	for name := range visited {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
		grape := grapeMap[name]
		for _, dep := range grape.Deps {
			edges[dep] = append(edges[dep], name)
			inDegree[name]++
		}
	}

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		next := edges[current]
		sort.Strings(next)
		for _, name := range next {
			inDegree[name]--
			if inDegree[name] == 0 {
				queue = append(queue, name)
				sort.Strings(queue)
			}
		}
	}

	if len(sorted) != len(visited) {
		return nil, fmt.Errorf("circular dependency: %s", findCycle(visited, grapeMap))
	}

	result := make([]*parser.GrapeFile, len(sorted))
	for i, name := range sorted {
		result[i] = grapeMap[name]
	}
	return result, nil
}

func findCycle(visited map[string]bool, grapeMap map[string]*parser.GrapeFile) string {
	for start := range visited {
		path := []string{start}
		seen := map[string]bool{start: true}
		if dfs(start, path, seen, grapeMap, &path) {
			last := path[len(path)-1]
			cycleStart := -1
			for i, node := range path[:len(path)-1] {
				if node == last {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := path[cycleStart:]
				cycle = append(cycle, last)
				result := ""
				for i, node := range cycle {
					if i > 0 {
						result += " -> "
					}
					result += node
				}
				return result
			}
		}
	}
	return "unknown cycle"
}

func dfs(current string, path []string, seen map[string]bool, grapeMap map[string]*parser.GrapeFile, result *[]string) bool {
	grape := grapeMap[current]
	for _, dep := range grape.Deps {
		if seen[dep] {
			*result = append(path, dep)
			return true
		}
		seen[dep] = true
		newPath := append(path, dep)
		if dfs(dep, newPath, seen, grapeMap, result) {
			return true
		}
		delete(seen, dep)
	}
	return false
}
