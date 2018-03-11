package component

import (
	"path/filepath"
	"strings"

	"github.com/bryanl/woowoo/ksutil"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type libPather interface {
	LibPath(envName string) (string, error)
}

// ParamOptions is options for parameters.
type ParamOptions struct {
	Index int
}

// Summary summarizes items found in components.
type Summary struct {
	ComponentName string
	IndexStr      string
	Index         int
	Type          string
	APIVersion    string
	Kind          string
	Name          string
}

// GVK converts a summary to a group - version - kind.
func (s *Summary) typeSpec() (*TypeSpec, error) {
	return NewTypeSpec(s.APIVersion, s.Kind)
}

// Component is a ksonnet Component interface.
type Component interface {
	Name() string
	Objects(paramsStr string) ([]*unstructured.Unstructured, error)
	SetParam(path []string, value interface{}, options ParamOptions) error
	DeleteParam(path []string, options ParamOptions) error
	Params() ([]NamespaceParameter, error)
	Summarize() ([]Summary, error)
}

const (
	// componentsDir is the name of the directory which houses components.
	componentsRoot = "components"
	// paramsFile is the params file for a component namespace.
	paramsFile = "params.libsonnet"
)

// Path returns returns the file system path for a component.
func Path(app ksutil.SuperApp, name string) (string, error) {
	ns, localName := ExtractNamespacedComponent(app, name)

	fis, err := afero.ReadDir(app.Fs(), ns.Dir())
	if err != nil {
		return "", err
	}

	var fileName string
	files := make(map[string]bool)

	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		base := strings.TrimSuffix(fi.Name(), filepath.Ext(fi.Name()))
		if _, ok := files[base]; ok {
			return "", errors.Errorf("Found multiple component files with component name %q", name)
		}
		files[base] = true

		if base == localName {
			fileName = fi.Name()
		}
	}

	if fileName == "" {
		return "", errors.Errorf("No component name %q found", name)
	}

	return filepath.Join(ns.Dir(), fileName), nil
}

// ExtractComponent extracts a component from a path.
func ExtractComponent(app ksutil.SuperApp, path string) (Component, error) {
	ns, componentName := ExtractNamespacedComponent(app, path)
	members, err := ns.Components()
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		if componentName == member.Name() {
			return member, nil
		}
	}

	return nil, errors.Errorf("unable to find component %q", componentName)
}

func isComponentDir(fs afero.Fs, path string) (bool, error) {
	files, err := afero.ReadDir(fs, path)
	if err != nil {
		return false, errors.Wrapf(err, "read files in %s", path)
	}

	for _, file := range files {
		if file.Name() == paramsFile {
			return true, nil
		}
	}

	return false, nil
}

// MakePathsByNamespace creates a map of component paths categorized by namespace.
func MakePathsByNamespace(app ksutil.SuperApp, env string) (map[Namespace][]string, error) {
	paths, err := MakePaths(app, env)
	if err != nil {
		return nil, err
	}

	m := make(map[Namespace][]string)

	for i := range paths {
		prefix := app.Root() + "/components/"
		if strings.HasSuffix(app.Root(), "/") {
			prefix = app.Root() + "components/"
		}
		path := strings.TrimPrefix(paths[i], prefix)
		ns, _ := ExtractNamespacedComponent(app, path)
		if _, ok := m[ns]; !ok {
			m[ns] = make([]string, 0)
		}

		m[ns] = append(m[ns], paths[i])
	}

	return m, nil
}

// MakePaths creates a slice of component paths
func MakePaths(app ksutil.SuperApp, env string) ([]string, error) {
	cpl, err := newComponentPathLocator(app, env)
	if err != nil {
		return nil, errors.Wrap(err, "create component path locator")
	}

	return cpl.Locate()
}
