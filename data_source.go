package gwr

import (
	"github.com/uber-go/gwr/internal/marshaled"
	"github.com/uber-go/gwr/internal/meta"
	"github.com/uber-go/gwr/source"
)

// DefaultDataSources is default data sources registry which data sources are
// added to by the module-level Add* functions.  It is used by all of the
// protocol servers if no data sources are provided.
var DefaultDataSources *source.DataSources

func init() {
	DefaultDataSources = source.NewDataSources()
	metaNouns := meta.NewNounDataSource(DefaultDataSources)
	DefaultDataSources.Add(marshaled.NewDataSource(metaNouns, nil))
	DefaultDataSources.SetObserver(metaNouns)
}

// AddDataSource adds a data source to the default data sources registry.
func AddDataSource(ds source.DataSource) error {
	return DefaultDataSources.Add(ds)
}

// AddGenericDataSource adds a generic data source to the default data sources
// registry.
func AddGenericDataSource(gds source.GenericDataSource) error {
	mds := marshaled.NewDataSource(gds, nil)
	return DefaultDataSources.Add(mds)
}
