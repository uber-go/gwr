// Package tap provides a simple GWR data sources for tapping into arbitrary
// points in your program where it wasn't worth the effort to write a
// specialized data source.
//
// Currently a simple watchable-only emitter source is provided.
//
// TODO: coming soon: a simple sampled-source that also supports get.
package tap

import (
	"fmt"
	"text/template"

	"github.com/uber-go/gwr"
	"github.com/uber-go/gwr/source"
)

var defaultTemplate = template.Must(template.New("tap_text").Parse(`
{{- define "item" -}}
{{ . }}
{{- end -}}
`))

// Emitter provides a simple watchable data source with easy emission.
type Emitter struct {
	name    string
	tmpl    *template.Template
	watcher source.GenericDataWatcher
}

// NewEmitter creates an Emitter with a given name and text template; if the
// template is nil, than a default template which just uses the default textual
// representation is used.
//
// The given name will be prefixed with "/tap/" automatically.
//
// Any templated passed must define an "item" block.
func NewEmitter(name string, tmpl *template.Template) *Emitter {
	name = fmt.Sprintf("/tap/%s", name)
	if tmpl == nil {
		tmpl = defaultTemplate
	}
	return &Emitter{
		name: name,
		tmpl: tmpl,
	}
}

// AddEmitter creates an emitter source and adds it to the default gwr sources.
func AddEmitter(name string, tmpl *template.Template) *Emitter {
	tap := NewEmitter(name, tmpl)
	gwr.AddGenericDataSource(tap)
	return tap
}

// Name returns the full name of the emitter source; this will be
// "/tap/name_given_to_New_Emitter".
func (em *Emitter) Name() string {
	return em.name
}

// TextTemplate returns the template used to marshal items human friendily.
func (em *Emitter) TextTemplate() *template.Template {
	return em.tmpl
}

// SetWatcher sets the watcher at source addition time.
func (em *Emitter) SetWatcher(watcher source.GenericDataWatcher) {
	em.watcher = watcher
}

// Active retruns true if there are any active watchers.
func (em *Emitter) Active() bool {
	return em.watcher.Active()
}

// Emit emits item(s) to any active watchers.  Returns true if the watcher is
// (still) active.
func (em *Emitter) Emit(items ...interface{}) bool {
	if !em.watcher.Active() {
		return false
	}
	switch len(items) {
	case 0:
		return true
	case 1:
		return em.watcher.HandleItem(items[0])
	default:
		return em.watcher.HandleItems(items)
	}
}

// EmitBatch emits batch of items.  Returns true if the watcher is (still)
// active.
func (em *Emitter) EmitBatch(items []interface{}) bool {
	if !em.watcher.Active() {
		return false
	}
	return em.watcher.HandleItems(items)
}
