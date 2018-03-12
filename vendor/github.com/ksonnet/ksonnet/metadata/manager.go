// Copyright 2017 The kubecfg authors
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package metadata

import (
	"fmt"
	"os/user"
	"path"
	"path/filepath"

	"github.com/ksonnet/ksonnet/component"
	"github.com/ksonnet/ksonnet/env"
	"github.com/ksonnet/ksonnet/metadata/app"
	"github.com/ksonnet/ksonnet/metadata/registry"
	str "github.com/ksonnet/ksonnet/strings"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const (
	ksonnetDir      = ".ksonnet"
	registriesDir   = ksonnetDir + "/registries"
	libDir          = "lib"
	componentsDir   = "components"
	environmentsDir = "environments"
	vendorDir       = "vendor"

	// Files names.
	componentParamsFile = "params.libsonnet"
	baseLibsonnetFile   = "base.libsonnet"
	appYAMLFile         = "app.yaml"
	registryYAMLFile    = "registry.yaml"
	partsYAMLFile       = "parts.yaml"

	// ComponentsExtCodeKey is the ExtCode key for component imports
	ComponentsExtCodeKey = "__ksonnet/components"
	// EnvExtCodeKey is the ExtCode key for importing environment metadata
	EnvExtCodeKey = "__ksonnet/environments"
	// ParamsExtCodeKey is the ExtCode key for importing component parameters
	ParamsExtCodeKey = "__ksonnet/params"

	// User-level ksonnet directories.
	userKsonnetRootDir = ".ksonnet"
	pkgSrcCacheDir     = "src"
)

type manager struct {
	appFS afero.Fs

	// Application paths.
	rootPath         string
	ksonnetPath      string
	registriesPath   string
	libPath          string
	componentsPath   string
	environmentsPath string
	vendorPath       string

	componentParamsPath string
	baseLibsonnetPath   string
	appYAMLPath         string

	// User-level paths.
	userKsonnetRootPath string
	pkgSrcCachePath     string
}

func findManager(p string, appFS afero.Fs) (*manager, error) {
	var lastBase string
	currBase := p

	for {
		currPath := path.Join(currBase, ksonnetDir)
		exists, err := afero.Exists(appFS, currPath)
		if err != nil {
			return nil, err
		}
		if exists {
			m, err := newManager(currBase, appFS)
			if err != nil {
				return nil, err
			}

			app, err := m.App()
			if err != nil {
				return nil, err
			}

			if err = app.Init(); err != nil {
				return nil, errors.Wrap(err, "initialize app schema")
			}

			return m, nil
		}

		lastBase = currBase
		currBase = filepath.Dir(currBase)
		if lastBase == currBase {
			return nil, fmt.Errorf("No ksonnet application found")
		}
	}
}

func initManager(name, rootPath string, k8sSpecFlag, serverURI, namespace *string, incubatorReg registry.Manager, appFS afero.Fs) (*manager, error) {
	m, err := newManager(rootPath, appFS)
	if err != nil {
		return nil, errors.Wrap(err, "create manager")
	}

	// Retrieve `registry.yaml`.
	registryYAMLData, err := generateRegistryYAMLData(incubatorReg)
	if err != nil {
		return nil, err
	}

	// Generate data for `app.yaml`.
	appYAMLData, err := generateAppYAMLData(name, incubatorReg.MakeRegistryRefSpec())
	if err != nil {
		return nil, err
	}

	// Generate data for `base.libsonnet`.
	baseLibData := genBaseLibsonnetContent()

	// Initialize directory structure.
	if err := m.createAppDirTree(name, appYAMLData, baseLibData, incubatorReg); err != nil {
		return nil, err
	}

	// Initialize user dir structure.
	if err := m.createUserDirTree(); err != nil {
		return nil, errorOnCreateFailure(name, err)
	}

	// Initialize environment, and cache specification data.
	if serverURI != nil {
		err := m.CreateEnvironment(defaultEnvName, *serverURI, *namespace, *k8sSpecFlag)
		if err != nil {
			return nil, errorOnCreateFailure(name, err)
		}
	}

	// Write out `incubator` registry spec.
	registryPath := m.registryPath(incubatorReg)
	err = afero.WriteFile(m.appFS, registryPath, registryYAMLData, defaultFilePermissions)
	if err != nil {
		return nil, errorOnCreateFailure(name, err)
	}

	return m, nil
}

func newManager(rootPath string, appFS afero.Fs) (*manager, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	userRootPath := str.AppendToPath(usr.HomeDir, userKsonnetRootDir)

	m := &manager{
		appFS: appFS,

		// Application paths.
		rootPath:         rootPath,
		ksonnetPath:      str.AppendToPath(rootPath, ksonnetDir),
		registriesPath:   str.AppendToPath(rootPath, registriesDir),
		libPath:          str.AppendToPath(rootPath, libDir),
		componentsPath:   str.AppendToPath(rootPath, componentsDir),
		environmentsPath: str.AppendToPath(rootPath, environmentsDir),
		vendorPath:       str.AppendToPath(rootPath, vendorDir),

		componentParamsPath: str.AppendToPath(rootPath, componentsDir, componentParamsFile),
		baseLibsonnetPath:   str.AppendToPath(rootPath, environmentsDir, baseLibsonnetFile),
		appYAMLPath:         str.AppendToPath(rootPath, appYAMLFile),

		// User-level paths.
		userKsonnetRootPath: userRootPath,
		pkgSrcCachePath:     str.AppendToPath(userRootPath, pkgSrcCacheDir),
	}

	return m, nil
}

func (m *manager) Root() string {
	return m.rootPath
}

func (m *manager) App() (app.App, error) {
	return app.Load(m.appFS, m.rootPath)
}

func (m *manager) LibPaths() (envPath, vendorPath string) {
	return m.environmentsPath, m.vendorPath
}

func (m *manager) GetDestination(envName string) (env.Destination, error) {
	appEnv, err := m.GetEnvironment(envName)
	if err != nil {
		return env.Destination{}, err
	}

	return appEnv.Destination, nil
}

func (m *manager) createUserDirTree() error {
	dirPaths := []string{
		m.userKsonnetRootPath,
		m.pkgSrcCachePath,
	}

	for _, p := range dirPaths {
		if err := m.appFS.MkdirAll(p, defaultFolderPermissions); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) createAppDirTree(name string, appYAMLData, baseLibData []byte, gh registry.Manager) error {
	exists, err := afero.DirExists(m.appFS, m.rootPath)
	if err != nil {
		return fmt.Errorf("Could not check existence of directory '%s':\n%v", m.rootPath, err)
	} else if exists {
		return fmt.Errorf("Could not create app; directory '%s' already exists", m.rootPath)
	}

	dirPaths := []string{
		m.rootPath,
		m.ksonnetPath,
		m.registriesPath,
		m.libPath,
		m.componentsPath,
		m.environmentsPath,
		m.vendorPath,
		m.registryDir(gh),
	}

	for _, p := range dirPaths {
		log.Debugf("Creating directory '%s'", p)
		if err := m.appFS.MkdirAll(p, defaultFolderPermissions); err != nil {
			return errorOnCreateFailure(name, err)
		}
	}

	filePaths := []struct {
		path    string
		content []byte
	}{
		{
			m.componentParamsPath,
			component.GenParamsContent(),
		},
		{
			m.baseLibsonnetPath,
			genBaseLibsonnetContent(),
		},
		{
			m.appYAMLPath,
			appYAMLData,
		},
		{
			m.baseLibsonnetPath,
			baseLibData,
		},
	}

	for _, f := range filePaths {
		log.Debugf("Creating file '%s'", f.path)
		if err := afero.WriteFile(m.appFS, f.path, f.content, defaultFilePermissions); err != nil {
			return err
		}
	}

	return nil
}

func generateRegistryYAMLData(incubatorReg registry.Manager) ([]byte, error) {
	regSpec, err := incubatorReg.FetchRegistrySpec()
	if err != nil {
		return nil, err
	}

	return regSpec.Marshal()
}

func generateAppYAMLData(name string, refs ...*app.RegistryRefSpec) ([]byte, error) {
	content := app.Spec{
		APIVersion:   app.DefaultAPIVersion,
		Kind:         app.Kind,
		Name:         name,
		Version:      app.DefaultVersion,
		Registries:   app.RegistryRefSpecs{},
		Environments: app.EnvironmentSpecs{},
	}

	for _, ref := range refs {
		err := content.AddRegistryRef(ref)
		if err != nil {
			return nil, err
		}
	}

	return content.Marshal()
}

func genBaseLibsonnetContent() []byte {
	return []byte(`local components = std.extVar("` + ComponentsExtCodeKey + `");
components + {
  // Insert user-specified overrides here.
}
`)
}

func errorOnCreateFailure(appName string, err error) error {
	return fmt.Errorf("%s\nTo undo this simply delete directory '%s' and re-run `ks init`.\nIf the error persists, try using flag '--context' to set a different context or run `ks init --help` for more options", err, appName)
}
