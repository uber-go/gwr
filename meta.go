package gwr

import "text/template"

const metaNounName = "/meta/nouns"

var nounsTextTemplate = template.Must(template.New("meta_nouns_text").Parse(`
{{- define "get" -}}
{{ range $name, $info := . -}}
- {{ $name }} formats: {{ $info.Formats }}
{{ end -}}
{{- end -}}
`))

type dataSourceUpdate struct {
	Type string                 `json:"type"`
	Info map[string]interface{} `json:"info"`
}

type metaNounDataSource struct {
	sources *DataSources
	watcher GenericDataWatcher
}

func (nds *metaNounDataSource) Name() string {
	return metaNounName
}

func (nds *metaNounDataSource) Attrs() map[string]interface{} {
	return nil
}

func (nds *metaNounDataSource) TextTemplate() *template.Template {
	return nounsTextTemplate
}

func (nds *metaNounDataSource) Get() interface{} {
	sources := nds.sources.sources
	info := make(map[string]interface{}, len(sources))
	for name, ds := range sources {
		info[name] = dsInfo(ds)
	}
	return info
}

func (nds *metaNounDataSource) GetInit() interface{} {
	return nds.Get()
}

func (nds *metaNounDataSource) Watch(watcher GenericDataWatcher) {
	nds.watcher = watcher
}

func (nds *metaNounDataSource) dataSourceAdded(ds DataSource) {
	if nds.watcher != nil {
		nds.watcher(dataSourceUpdate{"add", dsInfo(ds)})
	}
}

func dsInfo(ds DataSource) map[string]interface{} {
	return map[string]interface{}{
		"formats": ds.Formats(),
		"attrs":   ds.Attrs(), // TODO: flatten?
	}
}
