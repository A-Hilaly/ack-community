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

package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	awssdkmodel "github.com/aws/aws-sdk-go/private/model/api"
	"github.com/gertd/go-pluralize"

	"github.com/aws/aws-controllers-k8s/pkg/names"
)

type CRDOps struct {
	Create   *awssdkmodel.Operation
	ReadOne  *awssdkmodel.Operation
	ReadMany *awssdkmodel.Operation
	Update   *awssdkmodel.Operation
	Delete   *awssdkmodel.Operation
}

// CRDField represents a single field in the CRD's Spec or Status objects
type CRDField struct {
	CRD    *CRD
	Names  names.Names
	GoType string
	Shape  *awssdkmodel.Shape
}

// newCRDField returns a pointer to a new CRDField object
func newCRDField(
	crd *CRD,
	fieldNames names.Names,
	shape *awssdkmodel.Shape,
) *CRDField {
	// There are shapes that are called things like DBProxyStatus that are
	// fields in a DBProxy CRD... we need to ensure the type names don't
	// conflict.
	var gt string
	if shape.Type == "structure" {
		typeNames := names.New(shape.ShapeName)
		gt = "*" + typeNames.Camel
		if typeNames.Camel == crd.Kind+"Spec" || typeNames.Camel == crd.Kind+"Status" {
			gt = "*" + typeNames.Camel + "_SDK"
		}
	} else if shape.Type == "list" {
		// If it's a list type, where the element is a structure, we need to
		// set the GoType to the cleaned-up Camel-cased name
		typeNames := names.New(shape.GoTypeElem())
		gt = "[]*" + typeNames.Camel
		if typeNames.Camel == crd.Kind+"Spec" || typeNames.Camel == crd.Kind+"Status" {
			gt = "[]*" + typeNames.Camel + "_SDK"
		}
	} else {
		gt = shape.GoType()
	}
	return &CRDField{
		CRD:    crd,
		Names:  fieldNames,
		Shape:  shape,
		GoType: gt,
	}
}

type CRD struct {
	helper *Helper
	Names  names.Names
	Kind   string
	Plural string
	Ops    CRDOps
	// SpecFields is a map, keyed by the **original SDK member name** of
	// CRDField objects representing those fields in the CRD's Spec struct
	// field.
	SpecFields map[string]*CRDField
	// StatusFields is a map, keyed by the **original SDK member name** of
	// CRDField objects representing those fields in the CRD's Status struct
	// field. Note that there are no fields in StatusFields that are also in
	// SpecFields.
	StatusFields map[string]*CRDField
	SDKMapper    *SDKMapper
	// TypeImports is a map, keyed by an import string, with the map value
	// being the import alias
	TypeImports map[string]string
}

// AddSpecField adds a new CRDField of a given name and shape into the Spec
// field of a CRD
func (r *CRD) AddSpecField(
	memberNames names.Names,
	shape *awssdkmodel.Shape,
) {
	crdField := newCRDField(r, memberNames, shape)
	r.SpecFields[memberNames.Original] = crdField
}

// AddStatusField adds a new CRDField of a given name and shape into the Status
// field of a CRD
func (r *CRD) AddStatusField(
	memberNames names.Names,
	shape *awssdkmodel.Shape,
) {
	goPkgType := shape.GoTypeWithPkgNameElem()
	if strings.Contains(goPkgType, ".") {
		if strings.HasPrefix(goPkgType, "[]") {
			// For slice types, we just want the element type...
			goPkgType = goPkgType[2:]
		}
		if strings.HasPrefix(goPkgType, "*") {
			// For slice types, the element type might be a pointer to
			// a struct...
			goPkgType = goPkgType[1:]
		}
		pkg := strings.Split(goPkgType, ".")[0]
		if pkg != r.helper.sdkAPI.PackageName() {
			// Shape.GoPTypeWithPkgNameElem() always returns the type
			// as a full package dot-notation name. We only want to add
			// imports for "normal" package types like "time.Time", not
			// "ecr.ImageScanningConfiguration"
			r.AddTypeImport(pkg, "")
		}
	}
	crdField := newCRDField(r, memberNames, shape)
	r.StatusFields[memberNames.Original] = crdField
}

// AddTypeImport adds an entry in the CRD's TypeImports map for an import line
// and optional alias
func (r *CRD) AddTypeImport(
	packagePath string,
	alias string,
) {
	if r.TypeImports == nil {
		r.TypeImports = map[string]string{}
	}
	r.TypeImports[packagePath] = alias
}

// SpecFieldNames returns a sorted slice of field names for the Spec fields
func (r *CRD) SpecFieldNames() []string {
	res := make([]string, 0, len(r.SpecFields))
	for fieldName := range r.SpecFields {
		res = append(res, fieldName)
	}
	sort.Strings(res)
	return res
}

// GoCodeSetInput returns the Go code that sets an input shape's member fields
// from a CRD's Spec fields.
//
// We loop through the Spec fields, outputting code that looks something like
// this:
//
//   res.SetAttributes(r.ko.Spec.Attributes)
//   res.SetName(*r.ko.Spec.Name)
//   tmp0 := []*svcsdk.Tag{}
//   res.Tags = tmp0
func (r *CRD) GoCodeSetInput(
	opType OpType,
	inVarName string,
	koVarAccessor string,
	indentLevel int,
) string {
	var op *awssdkmodel.Operation
	switch opType {
	case OpTypeCreate:
		op = r.Ops.Create
	case OpTypeGet:
		op = r.Ops.ReadOne
	case OpTypeList:
		op = r.Ops.ReadMany
	case OpTypeUpdate:
		op = r.Ops.Update
	case OpTypeDelete:
		op = r.Ops.Delete
	default:
		return ""
	}
	if op == nil {
		return ""
	}
	inputShape := op.InputRef.Shape
	if inputShape == nil {
		return ""
	}

	out := "\n"
	tmpVarCount := 0
	tmpVarName := ""
	indent := strings.Repeat("\t", indentLevel)

	for _, fieldName := range r.SpecFieldNames() {
		specField := r.SpecFields[fieldName]
		memberShapeRef, found := inputShape.MemberRefs[specField.Names.Original]
		if !found || memberShapeRef.Shape == nil {
			continue
		}

		memberShape := memberShapeRef.Shape
		inAccessor := inVarName + "." + specField.Names.Original
		switch memberShape.Type {
		case "structure":
			tmpVarName = fmt.Sprintf("tmp%d", tmpVarCount)
			tmpVarCount++
			memberType := memberShape.GoTypeWithPkgName()
			memberType = r.replacePkgName(memberType, "svcsdk", false)
			out += fmt.Sprintf("%s%s := &%s{}\n", indent, tmpVarName, memberType)
			// TODO(jaypipes): Populate the struct's subfields recursively
			out += fmt.Sprintf("%s%s = %s\n", indent, inAccessor, tmpVarName)
		case "list":
			tmpVarName = fmt.Sprintf("tmp%d", tmpVarCount)
			tmpVarCount++
			// Trim off the [] prefix...
			memberType := memberShape.GoTypeWithPkgName()[2:]
			memberType = r.replacePkgName(memberType, "svcsdk", true)
			out += fmt.Sprintf("%s%s := []%s{}\n", indent, tmpVarName, memberType)
			// TODO(jaypipes): For each element in the source slice, append an
			// element to the target slice
			out += fmt.Sprintf("%s%s = %s\n", indent, inAccessor, tmpVarName)
		default:
			setTo := koVarAccessor + "." + specField.Names.Camel
			if memberShapeRef.UseIndirection() {
				setTo = "*" + setTo
			}
			out += fmt.Sprintf("%s%s.Set%s(%s)\n", indent, inVarName, specField.Names.Original, setTo)
		}
	}
	return out
}

// GoCodeSetOutput returns the Go code that sets a CRD's Status field value to
// the value of an output shape's member fields
//
// We loop through the output shape's fields, outputting code that looks something like
// this:
//
//   tmp0 := &ImageData{}
//   tmp0.Location = resp.ImageData.Location
//   tmp0.Tag = resp.ImageData.Tag
//   r.ko.Status.ImageData = tmp0
//   r.ko.Status.Name = resp.Name
//
func (r *CRD) GoCodeSetOutput(
	opType OpType,
	outVarAccessor string,
	koVarName string,
	indentLevel int,
) string {
	var op *awssdkmodel.Operation
	switch opType {
	case OpTypeCreate:
		op = r.Ops.Create
	case OpTypeGet:
		op = r.Ops.ReadOne
	case OpTypeList:
		op = r.Ops.ReadMany
	case OpTypeUpdate:
		op = r.Ops.Update
	case OpTypeDelete:
		op = r.Ops.Delete
	default:
		return ""
	}
	if op == nil {
		return ""
	}
	outputShape := op.OutputRef.Shape
	if outputShape == nil {
		return ""
	}

	// We might be in a "wrapper" shape. Unwrap it to find the real object
	// representation for the CRD's createOp. If there is a single member
	// shape and that member shape is a structure, unwrap it.
	if outputShape.UsedAsOutput && len(outputShape.MemberRefs) == 1 {
		for _, memberRef := range outputShape.MemberRefs {
			if memberRef.Shape.Type == "structure" {
				outVarAccessor += "." + memberRef.Shape.ShapeName
				outputShape = memberRef.Shape
			}
		}
	}
	out := "\n"
	tmpVarCount := 0
	tmpVarName := ""
	indent := strings.Repeat("\t", indentLevel)

	for _, fieldName := range outputShape.MemberNames() {
		statusField, found := r.StatusFields[fieldName]
		if !found {
			// TODO(jaypipes): Handle the special case of ARN for primary
			// resource identifier
			continue
		}
		memberShapeRef := outputShape.MemberRefs[fieldName]
		if memberShapeRef.Shape == nil {
			continue
		}

		memberShape := memberShapeRef.Shape
		outAccessor := koVarName + "." + statusField.Names.Camel
		switch memberShape.Type {
		case "structure":
			tmpVarName = fmt.Sprintf("tmp%d", tmpVarCount)
			tmpVarCount++
			memberType := memberShape.GoTypeWithPkgName()
			memberType = r.replacePkgName(memberType, "svcapitypes", false)
			out += fmt.Sprintf("%s%s := &%s{}\n", indent, tmpVarName, memberType)
			// TODO(jaypipes): Populate the struct's subfields recursively
			out += fmt.Sprintf("%s%s = %s\n", indent, outAccessor, tmpVarName)
		case "list":
			tmpVarName = fmt.Sprintf("tmp%d", tmpVarCount)
			tmpVarCount++
			// Trim off the [] prefix...
			memberType := memberShape.GoTypeWithPkgName()[2:]
			memberType = r.replacePkgName(memberType, "svcapitypes", true)
			out += fmt.Sprintf("%s%s := []%s{}\n", indent, tmpVarName, memberType)
			// TODO(jaypipes): For each element in the source slice, append an
			// element to the target slice
			out += fmt.Sprintf("%s%s = %s\n", indent, outAccessor, tmpVarName)
		default:
			setAccessor := koVarName + "." + statusField.Names.Camel
			setTo := outVarAccessor + "." + fieldName
			out += fmt.Sprintf("%s%s = %s\n", indent, setAccessor, setTo)
		}
	}
	return out
}

// replacePkgName accepts a type string, as returned by
// Shape.GoTypeWithPkgName() and replaces the package name of the aws-sdk-go
// SDK API (e.g. "ecr" for the ECR API) with the string "svcsdkapi" which is
// the only alias we always use in our templated output.
func (r *CRD) replacePkgName(
	subject string,
	replacePkgAlias string,
	keepPointer bool,
) string {
	memberType := subject
	// We need to convert any package name that the aws-sdk-private
	// model uses "such as 'ecr.' to just 'svcapitypes' since we always
	// alias the Kubernetes API types for the service API with that
	if strings.Contains(memberType, ".") {
		pkgName := strings.Split(memberType, ".")[0]
		typeName := strings.Split(memberType, ".")[1]
		apiPkgName := r.helper.sdkAPI.PackageName()
		if pkgName == apiPkgName || pkgName == "*"+apiPkgName {
			memberType = replacePkgAlias + "." + typeName
		} else {
			// Leave package prefixes like "time." alone...
			memberType = pkgName + "." + typeName
		}
		if strings.HasPrefix(pkgName, "*") && keepPointer {
			// Make sure to preserve pointer types...
			memberType = "*" + memberType
		}
	}
	return memberType
}

func newCRD(
	helper *Helper,
	crdNames names.Names,
	crdOps CRDOps,
) *CRD {
	pluralize := pluralize.NewClient()
	kind := crdNames.Camel
	plural := pluralize.Plural(kind)
	return &CRD{
		helper:       helper,
		Names:        crdNames,
		Kind:         kind,
		Plural:       plural,
		Ops:          crdOps,
		SpecFields:   map[string]*CRDField{},
		StatusFields: map[string]*CRDField{},
	}
}

var (
	ErrNilShapePointer = errors.New("found nil Shape pointer")
)

func (h *Helper) GetCRDs() ([]*CRD, error) {
	if h.crds != nil {
		return h.crds, nil
	}
	crds := []*CRD{}

	opMap := h.GetOperationMap()

	createOps := (*opMap)[OpTypeCreate]
	readOneOps := (*opMap)[OpTypeGet]
	readManyOps := (*opMap)[OpTypeList]
	updateOps := (*opMap)[OpTypeUpdate]
	deleteOps := (*opMap)[OpTypeDelete]

	for crdName, createOp := range createOps {
		crdNames := names.New(crdName)
		crdOps := CRDOps{
			Create:   createOps[crdName],
			ReadOne:  readOneOps[crdName],
			ReadMany: readManyOps[crdName],
			Update:   updateOps[crdName],
			Delete:   deleteOps[crdName],
		}
		crd := newCRD(h, crdNames, crdOps)
		sdkMapper := NewSDKMapper(crd)
		crd.SDKMapper = sdkMapper

		// OK, begin to gather the CRDFields that will go into the Spec struct.
		// These fields are those members of the Create operation's Input
		// Shape.
		inputShape := createOp.InputRef.Shape
		if inputShape == nil {
			return nil, ErrNilShapePointer
		}
		for memberName, memberShapeRef := range inputShape.MemberRefs {
			memberNames := names.New(memberName)
			if memberShapeRef.Shape == nil {
				return nil, ErrNilShapePointer
			}
			crd.AddSpecField(memberNames, memberShapeRef.Shape)
		}

		// Now process the fields that will go into the Status struct. We want
		// fields that are in the Create operation's Output Shape but that are
		// not in the Input Shape.
		outputShape := createOp.OutputRef.Shape
		if outputShape.UsedAsOutput && len(outputShape.MemberRefs) == 1 {
			// We might be in a "wrapper" shape. Unwrap it to find the real object
			// representation for the CRD's createOp. If there is a single member
			// shape and that member shape is a structure, unwrap it.
			for _, memberRef := range outputShape.MemberRefs {
				if memberRef.Shape.Type == "structure" {
					outputShape = memberRef.Shape
				}
			}
		}
		for memberName, memberShapeRef := range outputShape.MemberRefs {
			memberNames := names.New(memberName)
			if memberShapeRef.Shape == nil {
				return nil, ErrNilShapePointer
			}
			if _, found := crd.SpecFields[memberName]; found {
				// We don't put fields that are already in the Spec struct into
				// the Status struct
				continue
			}
			if strings.EqualFold(memberName, "arn") ||
				strings.EqualFold(memberName, crdName+"arn") {
				// Normalize primary resource ARN field in the returned output
				// shape. We want to map this Shape into the
				// Status.ACKResourceMetadata.ARN field
				sdkMapper.SetPrimaryResourceARNField(createOp, memberName)
				continue
			}
			crd.AddStatusField(memberNames, memberShapeRef.Shape)
		}

		crds = append(crds, crd)
	}
	sort.Slice(crds, func(i, j int) bool {
		return crds[i].Names.Camel < crds[j].Names.Camel
	})
	h.crds = crds
	return crds, nil
}

// GetOperationMap returns a map, keyed by the operation type and operation
// ID/name, of aws-sdk-go private/model/api.Operation struct pointers
func (h *Helper) GetOperationMap() *OperationMap {
	if h.opMap != nil {
		return h.opMap
	}
	// create an index of Operations by operation types and resource name
	opMap := OperationMap{}
	for opID, op := range h.sdkAPI.Operations {
		opType, resName := GetOpTypeAndResourceNameFromOpID(opID)
		if _, found := opMap[opType]; !found {
			opMap[opType] = map[string]*awssdkmodel.Operation{}
		}
		opMap[opType][resName] = op
	}
	h.opMap = &opMap
	return &opMap
}
