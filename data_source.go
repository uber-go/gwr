package gwr

import (
	"fmt"
	"io"
)

// DefaultDataSources is default data sources registry which data sources are
// added to by the module-level Add* functions.  It is used by all of the
// protocol servers if no data sources are provided.
var DefaultDataSources DataSources

func init() {
	DefaultDataSources.init()
}

// AddDataSource adds a data source to the default data sources registry.
func AddDataSource(ds DataSource) error {
	return DefaultDataSources.AddDataSource(ds)
}

// AddMarshaledDataSource adds a generically marshaled data source to the
// default data sources registry.
func AddMarshaledDataSource(gds GenericDataSource) error {
	return DefaultDataSources.AddMarshaledDataSource(gds)
}

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
	Name    string                 `json:"name"`
	Formats []string               `json:"formats"`
	Attrs   map[string]interface{} `json:"attrs"`
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
	dss := &DataSources{}
	dss.init()
	return dss
}

func (dss *DataSources) init() {
	dss.sources = make(map[string]DataSource, 2)
	dss.metaNouns.sources = dss
	dss.AddMarshaledDataSource(&dss.metaNouns)
}

// Get returns the named data source or nil if none is defined.
func (dss *DataSources) Get(name string) DataSource {
	source, ok := dss.sources[name]
	if ok {
		return source
	}
	return nil
}

// Info returns a map of all DataSource.Info() data
func (dss *DataSources) Info() map[string]DataSourceInfo {
	info := make(map[string]DataSourceInfo, len(dss.sources))
	for name, ds := range dss.sources {
		info[name] = ds.Info()
	}
	return info
}

// AddMarshaledDataSource adds a generically-marshaled data source. It is a
// convenience for AddDataSource(NewMarshaledDataSource(gds, nil))
func (dss *DataSources) AddMarshaledDataSource(gds GenericDataSource) error {
	mds := NewMarshaledDataSource(gds, nil)
	// TODO: useful to return mds?
	return dss.AddDataSource(mds)
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
