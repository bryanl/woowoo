package component

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bryanl/woowoo/k8sutil"
	"github.com/ksonnet/ksonnet/metadata/app"

	"github.com/bryanl/woowoo/params"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Jsonnet is a component base on jsonnet.
type Jsonnet struct {
	app        app.App
	source     string
	paramsPath string
}

var _ Component = (*Jsonnet)(nil)

// NewJsonnet creates an instance of Jsonnet.
func NewJsonnet(a app.App, source, paramsPath string) *Jsonnet {
	return &Jsonnet{
		app:        a,
		source:     source,
		paramsPath: paramsPath,
	}
}

// Name is the name of this component.
func (j *Jsonnet) Name() string {
	base := filepath.Base(j.source)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (j *Jsonnet) vmImporter(envName string) (*jsonnet.MemoryImporter, error) {
	libPath, err := j.app.LibPath(envName)
	if err != nil {
		return nil, err
	}

	readString := func(path string) (string, error) {
		filename := filepath.Join(libPath, path)
		var b []byte

		b, err = afero.ReadFile(j.app.Fs(), filename)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}

	dataK, err := readString("k.libsonnet")
	if err != nil {
		return nil, err
	}
	dataK8s, err := readString("k8s.libsonnet")
	if err != nil {
		return nil, err
	}

	importer := &jsonnet.MemoryImporter{
		Data: map[string]string{
			"k.libsonnet":   dataK,
			"k8s.libsonnet": dataK8s,
		},
	}

	return importer, nil
}

func jsonWalk(obj interface{}) ([]interface{}, error) {
	switch o := obj.(type) {
	case map[string]interface{}:
		if o["kind"] != nil && o["apiVersion"] != nil {
			return []interface{}{o}, nil
		}
		ret := []interface{}{}
		for _, v := range o {
			children, err := jsonWalk(v)
			if err != nil {
				return nil, err
			}
			ret = append(ret, children...)
		}
		return ret, nil
	case []interface{}:
		ret := make([]interface{}, 0, len(o))
		for _, v := range o {
			children, err := jsonWalk(v)
			if err != nil {
				return nil, err
			}
			ret = append(ret, children...)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("Unexpected object structure: %T", o)
	}
}

// Objects converts jsonnet to a slice of apimachinery unstructured objects.
func (j *Jsonnet) Objects(paramsStr, envName string) ([]*unstructured.Unstructured, error) {
	importer, err := j.vmImporter(envName)
	if err != nil {
		return nil, err
	}

	vm := jsonnet.MakeVM()
	vm.Importer(importer)
	vm.ExtCode("__ksonnet/params", paramsStr)

	snippet, err := afero.ReadFile(j.app.Fs(), j.source)
	if err != nil {
		return nil, err
	}

	evaluated, err := vm.EvaluateSnippet(j.source, string(snippet))
	if err != nil {
		return nil, err
	}

	var top interface{}
	if err = json.Unmarshal([]byte(evaluated), &top); err != nil {
		return nil, err
	}

	objects, err := jsonWalk(top)
	if err != nil {
		return nil, err
	}

	ret := make([]runtime.Object, 0, len(objects))
	for _, object := range objects {
		data, err := json.Marshal(object)
		if err != nil {
			return nil, err
		}
		uns, _, err := unstructured.UnstructuredJSONScheme.Decode(data, nil, nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, uns)
	}

	return k8sutil.FlattenToV1(ret)
}

// SetParam set parameter for a component.
func (j *Jsonnet) SetParam(path []string, value interface{}, options ParamOptions) error {
	paramsData, err := j.readParams()
	if err != nil {
		return err
	}

	updatedParams, err := params.Set(path, paramsData, j.Name(), value, paramsComponentRoot)
	if err != nil {
		return err
	}

	if err = j.writeParams(updatedParams); err != nil {
		return err
	}

	return nil
}

// DeleteParam deletes a param.
func (j *Jsonnet) DeleteParam(path []string, options ParamOptions) error {
	paramsData, err := j.readParams()
	if err != nil {
		return err
	}

	updatedParams, err := params.Delete(path, paramsData, j.Name(), paramsComponentRoot)
	if err != nil {
		return err
	}

	if err = j.writeParams(updatedParams); err != nil {
		return err
	}

	return nil
}

// Params returns params for a component.
func (j *Jsonnet) Params() ([]NamespaceParameter, error) {
	paramsData, err := j.readParams()
	if err != nil {
		return nil, err
	}

	props, err := params.ToMap(j.Name(), paramsData, paramsComponentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not find components")
	}

	var params []NamespaceParameter
	for k, v := range props {
		vStr, err := j.paramValue(v)
		if err != nil {
			return nil, err
		}
		np := NamespaceParameter{
			Component: j.Name(),
			Key:       k,
			Index:     "0",
			Value:     vStr,
		}

		params = append(params, np)
	}

	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})

	return params, nil
}

func (j *Jsonnet) paramValue(v interface{}) (string, error) {
	switch v.(type) {
	default:
		s := fmt.Sprintf("%v", v)
		return s, nil
	case string:
		s := fmt.Sprintf("%v", v)
		return strconv.Quote(s), nil
	case map[string]interface{}, []interface{}:
		b, err := json.Marshal(&v)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}
}

// Summarize creates a summary for the component.
func (j *Jsonnet) Summarize() ([]Summary, error) {
	return []Summary{
		{
			ComponentName: j.Name(),
			IndexStr:      "0",
			Type:          "jsonnet",
		},
	}, nil
}

func (j *Jsonnet) readParams() (string, error) {
	b, err := afero.ReadFile(j.app.Fs(), j.paramsPath)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (j *Jsonnet) writeParams(src string) error {
	return afero.WriteFile(j.app.Fs(), j.paramsPath, []byte(src), 0644)
}
