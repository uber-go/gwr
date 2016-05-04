package source

// Info is a convenience info descriptor about a data source.
type Info struct {
	Formats []string               `json:"formats"`
	Attrs   map[string]interface{} `json:"attrs"`
}

// GetInfo returns a structure that contains format and other information about
// a given data source.
func GetInfo(ds DataSource) Info {
	return Info{
		Formats: ds.Formats(),
		Attrs:   ds.Attrs(),
	}
}

// Info returns a map of info about all sources.
func (dss *DataSources) Info() map[string]Info {
	info := make(map[string]Info, len(dss.sources))
	for name, ds := range dss.sources {
		info[name] = GetInfo(ds)
	}
	return info
}
