package gwr

import (
	"bytes"
	"encoding/json"
)

// LDJSONMarshal is the usual Line-Delimited JSON
var LDJSONMarshal = ldJSONMarshal(0)

type ldJSONMarshal int

// Marshal marhshals data through the standard json module
func (x ldJSONMarshal) Marshal(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// Frame appends the delimiter
func (x ldJSONMarshal) Frame(data interface{}) ([]byte, error) {
	json, err := x.Marshal(data)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(json)+1))
	buf.Write(json)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// FrameInit appends the delimiter
func (x ldJSONMarshal) FrameInit(data interface{}) ([]byte, error) {
	return x.Frame(data)
}

// JSONSeqMarshal is RFC 7464 (application/json-seq) encoding
var JSONSeqMarshal = jsonSeqMarshal(0)

type jsonSeqMarshal int

// Marshal marhshals data through the standard json module
func (x jsonSeqMarshal) Marshal(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// Frame appends the delimiter
func (x jsonSeqMarshal) Frame(data interface{}) ([]byte, error) {
	json, err := x.Marshal(data)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(json)+2))
	buf.WriteByte('\x1e')
	buf.Write(json)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// FrameInit appends the delimiter
func (x jsonSeqMarshal) FrameInit(data interface{}) ([]byte, error) {
	return x.Frame(data)
}
