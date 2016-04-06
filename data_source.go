package gwr

import (
	"fmt"
	"io"
)

// DataSource is the interface implemented by all
// data sources.
type DataSource interface {
	// NOTE: implementation is self encoding, but may abstract internally
	Info() DataSourceInfo
	Get(format string, w io.Writer) error
	Watch(format string, w io.Writer) error
}

// DataSourceInfo provides a description of each
// data source, such as name and supported formats.
type DataSourceInfo struct {
	Name    string
	Formats []string
	Attrs   map[string]interface{}
}

// DataSources is a flat collection of DataSources
// with a meta introspection data source.
type DataSources struct {
	sources   map[string]DataSource
	metaNouns metaNounDataSource
}

// NewDataSources creates a DataSources structure
// an sets up its "/meta/nouns" data source.
func NewDataSources() *DataSources {
	dss := &DataSources{
		sources: make(map[string]DataSource, 2),
	}
	dss.metaNouns.sources = dss
	dss.AddDataSource(NewMarshaledDataSource(
		&dss.metaNouns,
		map[string]GenericDataMarshal{
			"json": LDJSONMarshal,
			"text": nounsTextMarshal,
		},
	))
	return dss
}

// Info returns a map of all DataSource.Info() data
func (dss *DataSources) Info() map[string]DataSourceInfo {
	info := make(map[string]DataSourceInfo, len(dss.sources))
	for name, ds := range dss.sources {
		info[name] = ds.Info()
	}
	return info
}

// AddDataSource adds a DataSource, if none is
// already defined for the given name.
func (dss *DataSources) AddDataSource(ds DataSource) error {
	info := ds.Info()
	_, ok := dss.sources[info.Name]
	if ok {
		return fmt.Errorf("data source already defined")
	}
	dss.sources[info.Name] = ds
	dss.metaNouns.dataSourceAdded(ds)
	return nil
}

// TODO: do we really need to support removing data sources?  I can see the
// case for intermediaries perhaps, and suspect that is needed... but punting
// for now.
