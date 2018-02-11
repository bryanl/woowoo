package ksonnet

import (
	"strings"

	"github.com/blang/semver"
	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
)

var (
	blockedReferences = []string{
		"io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta",
		"io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.JSONSchemaProps",
		"io.k8s.apimachinery.pkg.apis.meta.v1.Status",
	}

	blockedPropertyNames = []string{
		"status",
		"$ref",
		"$schema",
		"JSONSchemas",
		"apiVersion",
		"kind",
	}
)

// ExtractFn is a function which extracts properties from a schema.
type ExtractFn func(*Catalog, map[string]spec.Schema) (map[string]Property, error)

// CatalogOpt is an option for configuring Catalog.
type CatalogOpt func(*Catalog)

// CatalogOptExtractProperties is a Catalog option for setting the property
// extractor.
func CatalogOptExtractProperties(fn ExtractFn) CatalogOpt {
	return func(c *Catalog) {
		c.extractFn = fn
	}
}

// Catalog is a catalog definitions
type Catalog struct {
	apiSpec    *spec.Swagger
	extractFn  ExtractFn
	apiVersion semver.Version

	// memos
	typesCache  []Type
	fieldsCache []Field
}

// NewCatalog creates an instance of Catalog.
func NewCatalog(apiSpec *spec.Swagger, opts ...CatalogOpt) (*Catalog, error) {
	if apiSpec == nil {
		return nil, errors.New("apiSpec is nil")
	}

	if apiSpec.Info == nil {
		return nil, errors.New("apiSpec Info is nil")
	}

	parts := strings.SplitN(apiSpec.Info.Version, ".", 3)
	parts[0] = strings.TrimPrefix(parts[0], "v")
	vers := strings.Join(parts, ".")
	apiVersion, err := semver.Parse(vers)
	if err != nil {
		return nil, errors.Wrap(err, "invalid apiSpec version")
	}

	c := &Catalog{
		apiSpec:    apiSpec,
		extractFn:  extractProperties,
		apiVersion: apiVersion,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Version returns the Kubernetes API version represented by this Catalog.
func (c *Catalog) Version() string {
	return c.apiVersion.String()
}

// Types returns a slice of all types.
func (c *Catalog) Types() ([]Type, error) {
	if c.typesCache != nil {
		return c.typesCache, nil
	}

	var resources []Type

	for name, schema := range c.definitions() {
		desc, err := ParseDescription(name)
		if err != nil {
			return nil, errors.Wrapf(err, "parse description for %s", name)
		}

		component, err := NewComponent(schema)
		if err != nil {
			continue
		}

		props, err := c.extractFn(c, schema.Properties)
		if err != nil {
			return nil, errors.Wrapf(err, "extract propererties from %s", name)
		}

		kind := NewType(name, schema.Description, desc.Codebase, desc.Group, *component, props)

		resources = append(resources, kind)
	}

	c.typesCache = resources

	return resources, nil
}

// Fields returns a slice of all fields.
func (c *Catalog) Fields() ([]Field, error) {
	if c.fieldsCache != nil {
		return c.fieldsCache, nil
	}

	var types []Field

	for name, schema := range c.definitions() {
		desc, err := ParseDescription(name)
		if err != nil {
			return nil, errors.Wrapf(err, "parse description for %s", name)
		}

		props, err := c.extractFn(c, schema.Properties)
		if err != nil {
			return nil, errors.Wrapf(err, "extract propererties from %s", name)
		}
		t := NewField(name, schema.Description, desc.Codebase, desc.Group, desc.Version, desc.Kind, props)
		types = append(types, *t)
	}

	c.fieldsCache = types
	return types, nil
}

func (c *Catalog) isFormatRef(name string) (bool, error) {
	schema, ok := c.apiSpec.Definitions[name]
	if !ok {
		return false, errors.Errorf("%s was not found", name)
	}

	if schema.Format != "" {
		return true, nil
	}

	return false, nil
}

// Field returns a field by definition id. If the type cannot be found, it returns an error.
func (c *Catalog) Field(name string) (*Field, error) {
	types, err := c.Fields()
	if err != nil {
		return nil, err
	}

	for _, ty := range types {
		if ty.Identifier() == name {
			return &ty, nil
		}
	}

	return nil, errors.Errorf("%s was not found", name)
}

// Resource returns a resource by group, version, kind. If the field cannot be found,
// it returns an error
func (c *Catalog) Resource(group, version, kind string) (*Type, error) {
	resources, err := c.Types()
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		if group == resource.Group() &&
			version == resource.Version() &&
			kind == resource.Kind() {
			return &resource, nil
		}
	}

	return nil, errors.Errorf("unable to find %s.%s.%s",
		group, version, kind)
}

// TypesWithDescendant returns types who have the specificed definition as a descendant.
// This list doesn't not include List types (e.g. DeploymentList).
func (c *Catalog) TypesWithDescendant(definition string) ([]Type, error) {
	types, err := c.Types()
	if err != nil {
		return nil, errors.Wrap(err, "retrieve types")
	}

	var out []Type
	for _, ty := range types {

		if strings.HasSuffix(ty.Kind(), "List") {
			continue
		}
		tf, err := c.descend(definition, ty.Properties())
		if err != nil {
			return nil, err
		}

		if tf {
			out = append(out, ty)
		}
	}

	return out, nil
}

func (c *Catalog) descend(definition string, m map[string]Property) (bool, error) {
	for _, prop := range m {
		if ref := prop.Ref(); ref != "" {

			if ref == definition {
				return true, nil
			}

			f, err := c.Field(ref)
			if err != nil {
				return false, errors.Wrapf(err, "find field %s", ref)
			}

			tf, err := c.descend(definition, f.Properties())
			if err != nil {
				return false, err
			}

			if tf {
				return true, nil
			}
		}
	}

	return false, nil
}

func isValidDefinition(name string, ver semver.Version) bool {
	checkVer := semver.Version{Major: 1, Minor: 7}
	if ver.GTE(checkVer) {
		return !strings.HasPrefix(name, "io.k8s.kubernetes.pkg.api")
	}

	return true
}

// extractRef extracts a ref from a schema.
func extractRef(schema spec.Schema) string {
	return strings.TrimPrefix(schema.Ref.String(), "#/definitions/")
}

func (c *Catalog) definitions() spec.Definitions {
	out := spec.Definitions{}

	for name, schema := range c.apiSpec.Definitions {
		if isValidDefinition(name, c.apiVersion) {
			out[name] = schema
		}
	}

	return out
}
