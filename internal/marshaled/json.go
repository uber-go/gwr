package marshaled

import "encoding/json"

// LDJSONMarshal is the usual Line-Delimited JSON
var LDJSONMarshal = ldJSONMarshal(0)

type ldJSONMarshal int

// MarshalGet marhshals data through the standard json module.
func (x ldJSONMarshal) MarshalGet(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// MarshalInit marhshals data through the standard json module.
func (x ldJSONMarshal) MarshalInit(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// MarshalItem marhshals data through the standard json module.
func (x ldJSONMarshal) MarshalItem(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// FrameItem appends the newline record delimiter
func (x ldJSONMarshal) FrameItem(json []byte) ([]byte, error) {
	n := len(json)
	frame := make([]byte, n+1)
	copy(frame, json)
	frame[n] = '\n'
	return frame, nil
}
