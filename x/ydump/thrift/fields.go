// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package thrift

import (
	"sort"
	"strconv"

	"github.com/yarpc/yab/sorted"

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/wire"
)

type fields struct {
	exact, fieldIDs, fuzzy map[string]*compile.FieldSpec
}

func (fs fields) getField(name string) (*compile.FieldSpec, bool) {
	if f, ok := fs.exact[name]; ok {
		return f, true
	}
	if f, ok := fs.fieldIDs[name]; ok {
		return f, true
	}
	if f, ok := fs.fuzzy[fuzz(name)]; ok {
		return f, true
	}

	return nil, false
}

// fieldMap returns maps from string to the field spec.
// The first map is an exact map, which uses the name as specified in the Thrift file.
// The second map is a fuzzy map, which will ignore case, and ignore
// any non-alphanumberic characters.
func getFields(spec compile.FieldGroup) fields {
	exact := make(map[string]*compile.FieldSpec)
	fieldIDs := make(map[string]*compile.FieldSpec)
	fuzzy := make(map[string]*compile.FieldSpec)
	for _, f := range spec {
		exact[f.ThriftName()] = f
		fieldIDs[strconv.Itoa(int(f.ID))] = f
		fuzzy[fuzz(f.ThriftName())] = f
	}

	// If there's any field clashes in the fuzzy map, skip fuzzy matching.
	if len(exact) != len(fuzzy) {
		fuzzy = nil
	}

	return fields{
		exact:    exact,
		fieldIDs: fieldIDs,
		fuzzy:    fuzzy,
	}
}

func fieldGroupToValue(fieldsList compile.FieldGroup, request map[string]interface{}) ([]wire.Field, error) {
	var (
		fields = getFields(fieldsList)

		err = fieldGroupError{available: sorted.MapKeys(fields.exact)}

		// userFields is the user-specified values by field name.
		userFields = make(map[string]interface{})
	)

	for k, v := range request {
		field, ok := fields.getField(k)
		if !ok {
			err.addNotFound(k)
			continue
		}

		userFields[field.ThriftName()] = v
	}

	for k, arg := range fields.exact {
		if _, ok := userFields[arg.Name]; ok {
			continue
		}

		// Unspecified fields are always set to the Default value.
		if arg.Default != nil {
			userFields[arg.Name] = constToRequest(arg.Default)
			continue
		}

		if arg.Required {
			err.addMissingRequired(k)
			continue
		}
	}

	if err := err.asError(); err != nil {
		return nil, err
	}

	return fieldsMapToValue(fields.exact, userFields)
}

// fieldMapToValue converts the userFields to a list of wire.Field.
// It does not do any error checking.
func fieldsMapToValue(fields map[string]*compile.FieldSpec, userFields map[string]interface{}) ([]wire.Field, error) {
	wireFields := make([]wire.Field, 0, len(userFields))
	for k, userValue := range userFields {
		spec := fields[k]
		value, err := ToWireValue(spec.Type, userValue)
		if err != nil {
			return nil, err
		}

		wireFields = append(wireFields, wire.Field{
			ID:    spec.ID,
			Value: value,
		})
	}

	// Sort the fields so that we generate consistent data.
	sort.Sort(byFieldID(wireFields))
	return wireFields, nil
}

const upperToLower = 'a' - 'A'

// fuzz returns a copy of fieldName that is suitable for fuzzy matching.
// It lowercases the string, and removes non-alphanumeric characters.
func fuzz(fieldName string) string {
	newField := make([]byte, 0, len(fieldName))
	for i := 0; i < len(fieldName); i++ {
		c := fieldName[i]
		switch {
		case c >= 'A' && c <= 'Z':
			newField = append(newField, c+upperToLower)
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			newField = append(newField, c)
		}
	}

	return string(newField)
}

type byFieldID []wire.Field

func (p byFieldID) Len() int           { return len(p) }
func (p byFieldID) Less(i, j int) bool { return p[i].ID < p[j].ID }
func (p byFieldID) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
