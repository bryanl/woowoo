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

package kubecfg

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/ksonnet/ksonnet/client"
	"github.com/ksonnet/ksonnet/utils"
)

// ValidateCmd represents the validate subcommand
type ValidateCmd struct {
	ClientConfig *client.Config
	Env          string
}

func (c ValidateCmd) Run(apiObjects []*unstructured.Unstructured, out io.Writer) error {
	_, discovery, _, err := client.InitClient(c.Env)
	if err != nil {
		return err
	}
	hasError := false

	for _, obj := range apiObjects {
		desc := fmt.Sprintf("%s %s", utils.ResourceNameFor(discovery, obj), utils.FqName(obj))
		log.Info("Validating ", desc)

		var allErrs []error

		schema, err := utils.NewSwaggerSchemaFor(discovery, obj.GroupVersionKind().GroupVersion())
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("Unable to fetch schema: %v", err))
		} else {
			// Validate obj
			allErrs = append(allErrs, schema.Validate(obj)...)
		}

		for _, err := range allErrs {
			log.Errorf("Error in %s: %v", desc, err)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("Validation failed")
	}

	return nil
}
