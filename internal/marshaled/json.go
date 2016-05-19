// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
