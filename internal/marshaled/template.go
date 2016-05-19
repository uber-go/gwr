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

package marshaled

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplatedMarshal hooks together text/template to create a data source.
type TemplatedMarshal struct {
	tmpl                        *template.Template
	getName, initName, itemName string
}

// NewTemplatedMarshal creates a TemplatedMarshal from a template.  The default
// template names "get", "init", and "item" are used, if defined in the parsed
// template.
func NewTemplatedMarshal(tmpl *template.Template) *TemplatedMarshal {
	var getName, initName, itemName string
	if tmpl.Lookup("get") != nil {
		getName = "get"
	}
	if tmpl.Lookup("init") != nil {
		initName = "init"
	}
	if tmpl.Lookup("item") != nil {
		itemName = "item"
	}
	return &TemplatedMarshal{
		tmpl:     tmpl,
		getName:  getName,
		initName: initName,
		itemName: itemName,
	}
}

// TODO: need accessors for the names?

// MarshalGet returns the rendered bytes from the get template.  If no get
// template is defined, an error is returned.
func (tm *TemplatedMarshal) MarshalGet(data interface{}) ([]byte, error) {
	if len(tm.getName) == 0 {
		return nil, fmt.Errorf("only streaming is supported by the data format; no get template defined")
	}
	var buf bytes.Buffer
	if err := tm.tmpl.ExecuteTemplate(&buf, tm.getName, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalInit returns the rendered bytes from the init template.  If no init
// template is defined, an error is returned.
func (tm *TemplatedMarshal) MarshalInit(data interface{}) ([]byte, error) {
	if len(tm.initName) == 0 {
		return nil, fmt.Errorf("streaming is unsupported by the data format; no init template defined")
	}
	var buf bytes.Buffer
	if err := tm.tmpl.ExecuteTemplate(&buf, tm.initName, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalItem returns the rendered bytes from the item template.  If no item
// template is defined, an error is returned.
func (tm *TemplatedMarshal) MarshalItem(data interface{}) ([]byte, error) {
	if len(tm.itemName) == 0 {
		return nil, fmt.Errorf("streaming is unsupported by the data format; no item template defined")
	}
	var buf bytes.Buffer
	if err := tm.tmpl.ExecuteTemplate(&buf, tm.itemName, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FrameItem appends a newline
func (tm *TemplatedMarshal) FrameItem(json []byte) ([]byte, error) {
	n := len(json)
	frame := make([]byte, n+1)
	copy(frame, json)
	frame[n] = '\n'
	return frame, nil
}
