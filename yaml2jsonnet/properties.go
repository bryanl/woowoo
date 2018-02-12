package yaml2jsonnet

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// PropertyPath contains a property path.
type PropertyPath struct {
	Path  []string
	Value interface{}
}

// Properties are document properties
type Properties map[interface{}]interface{}

// Paths returns a list of paths in properties.
func (p Properties) Paths(gvk GVK) []PropertyPath {
	ch := make(chan PropertyPath)

	go func() {
		base := []string{gvk.Group, gvk.Version, gvk.Kind}
		iterateMap(ch, base, p)
		close(ch)
	}()

	var out []PropertyPath
	for pr := range ch {
		out = append(out, pr)
	}

	return out
}

func iterateMap(ch chan PropertyPath, base []string, m map[interface{}]interface{}) {
	localBase := make([]string, len(base))
	copy(localBase, base)

	var keys []interface{}
	for k := range m {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		a := keys[i].(string)
		b := keys[j].(string)

		return a < b
	})

	for i := range keys {
		name := keys[i].(string)
		switch t := m[name].(type) {
		default:
			panic(fmt.Sprintf("not sure what to do with %T", t))
		case map[interface{}]interface{}:
			newBase := append(localBase, name)
			iterateMap(ch, newBase, t)
		case string, int, []interface{}:
			ch <- PropertyPath{
				Path: append(base, name),
			}
		}
	}
}

// Value returns the value at a path.
func (p Properties) Value(path []string) (interface{}, error) {
	return valueSearch(path, p)
}

func valueSearch(path []string, m map[interface{}]interface{}) (interface{}, error) {
	var keys []interface{}
	for k := range m {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		a := keys[i].(string)
		b := keys[j].(string)

		return a < b
	})

	for i := range keys {
		name := keys[i].(string)
		if name == path[0] {

			switch t := m[name].(type) {
			default:
				panic(fmt.Sprintf("not sure what to do with %T", t))
			case map[interface{}]interface{}:
				if len(path) == 1 {
					return t, nil
				}
				return valueSearch(path[1:], t)
			case string, int, []interface{}:
				return t, nil
			}
		}
	}

	return nil, errors.Errorf("unable to find %s", strings.Join(path, "."))
}