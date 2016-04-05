package gwr

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type dataSourceUpdate struct {
	Type string
	Info DataSourceInfo
}

type metaNounDataSource struct {
	sources  *DataSources
	watchers []io.Writer
}

func (nds *metaNounDataSource) Info() DataSourceInfo {
	return DataSourceInfo{
		Name:    "/meta/nouns",
		Formats: []string{"JSON"},
		Attrs:   nil,
	}
}

func (nds *metaNounDataSource) Get(format string, w io.Writer) error {
	if strings.Compare(format, "JSON") != 0 {
		return fmt.Errorf("unsupported format")
	}
	return nds.writeFullInfo(w)
}

func (nds *metaNounDataSource) Watch(format string, w io.Writer) error {
	if strings.Compare(format, "JSON") != 0 {
		return fmt.Errorf("unsupported format")
	}

	if err := nds.writeFullInfo(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	nds.watchers = append(nds.watchers, w)

	return nil
}

func (nds *metaNounDataSource) writeFullInfo(w io.Writer) error {
	buf, err := json.Marshal(nds.sources.Info())
	if err != nil {
		return err
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}

func (nds *metaNounDataSource) dataSourceAdded(ds DataSource) {
	if len(nds.watchers) > 0 {
		nds.emitUpdate(dataSourceUpdate{"add", ds.Info()})
	}
}

func (nds *metaNounDataSource) emitUpdate(update dataSourceUpdate) {
	buf, err := json.Marshal(update)
	if err != nil {
		return // TODO err
	}

	// TODO: parallel write
	var failed []int
	for i, w := range nds.watchers {
		if _, err := w.Write(buf); err != nil {
			failed = append(failed, i)
		}
	}

	if len(failed) != 0 {
		// NOTE: parallel write above may invalidate the assumed failed
		// ordering, but for now we can get away with this (pre-)sort-merge
		var okay []io.Writer
		for i, w := range nds.watchers {
			if i != failed[0] {
				okay = append(okay, w)
			}
			if i >= failed[0] {
				failed = failed[1:]
				if len(failed) == 0 {
					if i < len(nds.watchers)-1 {
						okay = append(okay, nds.watchers[i+1:]...)
					}
					break
				}
			}
		}
		nds.watchers = okay
	}
}
