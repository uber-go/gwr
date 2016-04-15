package gwr

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

// FrameInit appends a newline
func (tm *TemplatedMarshal) FrameInit(json []byte) ([]byte, error) {
	return tm.FrameItem(json)
}

// FrameItem appends a newline
func (tm *TemplatedMarshal) FrameItem(json []byte) ([]byte, error) {
	n := len(json)
	frame := make([]byte, n+1)
	copy(frame, json)
	frame[n] = '\n'
	return frame, nil
}
