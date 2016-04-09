package gwr

import "text/template"

var nounsTextTemplate = template.Must(template.New("meta_nouns_text").Parse(`
{{- define "get" -}}
{{ range $name, $info := . -}}
- {{ $name }} formats: {{ $info.Formats }}
{{ end -}}
{{- end -}}
`))

type dataSourceUpdate struct {
	Type string
	Info DataSourceInfo
}

type metaNounDataSource struct {
	sources *DataSources
	watcher GenericDataWatcher
}

func (nds *metaNounDataSource) Info() GenericDataSourceInfo {
	return GenericDataSourceInfo{
		Name:         "/meta/nouns",
		Attrs:        nil,
		TextTemplate: nounsTextTemplate,
	}
}

func (nds *metaNounDataSource) Get() interface{} {
	return nds.sources.Info()
}

func (nds *metaNounDataSource) GetInit() interface{} {
	return nds.sources.Info()
}

func (nds *metaNounDataSource) Watch(watcher GenericDataWatcher) {
	nds.watcher = watcher
}

func (nds *metaNounDataSource) dataSourceAdded(ds DataSource) {
	if nds.watcher != nil {
		update := dataSourceUpdate{"add", ds.Info()}
		nds.watcher(update)
	}
}
