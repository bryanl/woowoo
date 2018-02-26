package component

import (
	"testing"

	"github.com/bryanl/woowoo/jsonnetutil"
	"github.com/stretchr/testify/require"
)

func TestValueExtractor_Extract(t *testing.T) {
	node, err := jsonnetutil.Import("testdata/k8s.libsonnet")
	require.NoError(t, err)

	props := Properties{
		"metadata": map[interface{}]interface{}{
			"name": "certificates.certmanager.k8s.io",
			"labels": map[interface{}]interface{}{
				"app":      "cert-manager",
				"chart":    "cert-manager-0.2.2",
				"release":  "cert-manager",
				"heritage": "Tiller",
			},
		},
		"spec": map[interface{}]interface{}{
			"group":   "certmanager.k8s.io",
			"version": "v1alpha1",
			"names": map[interface{}]interface{}{
				"kind":   "Certificate",
				"plural": "certificates",
			},
			"scope": "Namespaced",
		},
	}

	gvk := GVK{
		GroupPath: []string{
			"apiextensions.k8s.io",
		},
		Version: "v1beta1",
		Kind:    "customResourceDefinition",
	}

	ve := NewValueExtractor(node)
	got, err := ve.Extract(gvk, props)

	crd := "apiextensions.v1beta1.customResourceDefinition."

	expected := map[string]Values{
		crd + "mixin.metadata.labels": Values{
			Setter: crd + "mixin.metadata.withLabels",
			Value: map[interface{}]interface{}{
				"app":      "cert-manager",
				"chart":    "cert-manager-0.2.2",
				"release":  "cert-manager",
				"heritage": "Tiller",
			},
		},
		crd + "mixin.metadata.name": Values{
			Setter: crd + "mixin.metadata.withName",
			Value:  "certificates.certmanager.k8s.io",
		},
		crd + "mixin.spec.group": Values{
			Setter: crd + "mixin.spec.withGroup",
			Value:  "certmanager.k8s.io",
		},
		crd + "mixin.spec.names.kind": Values{
			Setter: crd + "mixin.spec.names.withKind",
			Value:  "Certificate",
		},
		crd + "mixin.spec.names.plural": Values{
			Setter: crd + "mixin.spec.names.withPlural",
			Value:  "certificates",
		},
		crd + "mixin.spec.scope": Values{
			Setter: crd + "mixin.spec.withScope",
			Value:  "Namespaced",
		},
		crd + "mixin.spec.version": Values{
			Setter: crd + "mixin.spec.withVersion",
			Value:  "v1alpha1",
		},
	}

	require.Equal(t, expected, got)
}