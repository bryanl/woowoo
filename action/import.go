package action

import (
	"bytes"
	"path/filepath"
	"strings"

	kscomponent "github.com/ksonnet/ksonnet/component"
	ksparam "github.com/ksonnet/ksonnet/metadata/params"
	"github.com/ksonnet/ksonnet/prototype"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Import imports files or directories into ksonnet.
func Import(fs afero.Fs, nsName, path string) error {
	i, err := newImport(fs, nsName, path)
	if err != nil {
		return err
	}

	return i.Run()
}

type componentImport struct {
	nsName string
	path   string

	*base
}

// newImport creates an instance of Show. `nsName` is the name of the component and
// entity is the file or directory to import.
func newImport(fs afero.Fs, nsName, path string) (*componentImport, error) {
	b, err := new(fs)
	if err != nil {
		return nil, err
	}

	i := &componentImport{
		nsName: nsName,
		path:   path,
		base:   b,
	}

	return i, nil
}

// Run runs the import process.
func (i *componentImport) Run() error {
	pathFi, err := i.app.Fs().Stat(i.path)
	if err != nil {
		return err
	}

	var paths []string
	if pathFi.IsDir() {
		fis, err := afero.ReadDir(i.app.Fs(), i.path)
		if err != nil {
			return err
		}

		for _, fi := range fis {
			path := filepath.Join(i.path, fi.Name())
			paths = append(paths, path)
		}
	} else {
		paths = append(paths, i.path)
	}

	for _, path := range paths {
		if err := i.importFile(path); err != nil {
			return err
		}
	}

	return nil
}

func (i *componentImport) importFile(fileName string) error {
	var name bytes.Buffer
	if i.nsName != "" {
		name.WriteString(i.nsName + "/")
	}

	base := filepath.Base(fileName)
	ext := filepath.Ext(base)

	templateType, err := prototype.ParseTemplateType(strings.TrimPrefix(ext, "."))
	if err != nil {
		return errors.Wrap(err, "parse template type")
	}

	name.WriteString(strings.TrimSuffix(base, ext))

	contents, err := afero.ReadFile(i.app.Fs(), fileName)
	if err != nil {
		return errors.Wrap(err, "read manifest")
	}

	params := ksparam.Params{}

	_, err = kscomponent.Create(i.app.Fs(), i.app.Root(), name.String(), string(contents), params, templateType)
	if err != nil {
		return errors.Wrap(err, "create component")
	}

	return nil
}
