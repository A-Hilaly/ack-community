// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package schema

import (
	"sort"

	"github.com/aws/aws-service-operator-k8s/pkg/model"
	"github.com/aws/aws-service-operator-k8s/pkg/names"
)

func (h *Helper) GetTypeDefs() ([]*model.TypeDef, error) {
	crds, err := h.GetCRDs()
	if err != nil {
		return nil, err
	}
	tdefs := []*model.TypeDef{}

	payloads := h.getPayloads()

	crdNames := []string{}
	for _, crd := range crds {
		crdNames = append(crdNames, crd.Kind)
	}

	for shapeName, shape := range h.sdkAPI.Shapes {
		if inStrings(shapeName, crdNames) {
			// CRDs are already top-level structs
			continue
		}
		if inStrings(shapeName, payloads) {
			// Payloads are not type defs
			continue
		}
		if shape.Type != "structure" {
			continue
		}
		if shape.Exception {
			// Neither are exceptions
			continue
		}
		attrs := map[string]*model.Attr{}
		for propName, memberRef := range shape.MemberRefs {
			propNames := names.New(propName)
			propShape := memberRef.Shape
			attrs[propName] = model.NewAttr(propNames, propShape.GoType(), propShape)
		}
		if len(attrs) == 0 {
			// Just ignore these...
			continue
		}
		tdefs = append(tdefs, &model.TypeDef{
			Names: names.New(shapeName),
			Attrs: attrs,
		})
	}
	sort.Slice(tdefs, func(i, j int) bool {
		return tdefs[i].Names.Camel < tdefs[j].Names.Camel
	})
	return tdefs, nil
}
