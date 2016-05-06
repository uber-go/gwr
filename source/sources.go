package source

import "fmt"

// DataSourcesObserver is an interface to observe data sources changes.
//
// Observation happens after after the source has been added (resp. removed).
type DataSourcesObserver interface {
	SourceAdded(ds DataSource)
	SourceRemoved(ds DataSource)
}

// DataSources is a flat collection of DataSources
// with a meta introspection data source.
type DataSources struct {
	sources map[string]DataSource
	obs     DataSourcesObserver
}

// NewDataSources creates a DataSources structure
// an sets up its "/meta/nouns" data source.
func NewDataSources() *DataSources {
	dss := &DataSources{
		sources: make(map[string]DataSource, 2),
	}
	return dss
}

// SetObserver sets the (single!) observer of data source changes; if nil is
// passed, observation is disabled.
func (dss *DataSources) SetObserver(obs DataSourcesObserver) {
	dss.obs = obs
}

// Get returns the named data source or nil if none is defined.
func (dss *DataSources) Get(name string) DataSource {
	source, ok := dss.sources[name]
	if ok {
		return source
	}
	return nil
}

// Add a DataSource, if none is already defined for the given name.
func (dss *DataSources) Add(ds DataSource) error {
	name := ds.Name()
	if _, ok := dss.sources[name]; ok {
		return fmt.Errorf("data source already defined")
	}
	dss.sources[name] = ds
	if dss.obs != nil {
		dss.obs.SourceAdded(ds)
	}
	return nil
}

// Remove a DataSource by name, if any exsits.  Returns the source removed, nil
// if none was defined.
func (dss *DataSources) Remove(name string) DataSource {
	ds, ok := dss.sources[name]
	if ok {
		delete(dss.sources, name)
		if dss.obs != nil {
			dss.obs.SourceRemoved(ds)
		}
	}
	return ds
}
