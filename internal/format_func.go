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

package internal

// FormatFunc is a convenience type to define simple GenericDataFormats.
type FormatFunc func(interface{}) ([]byte, error)

// MarshalGet calls the wrapped function.
func (ff FormatFunc) MarshalGet(item interface{}) ([]byte, error) {
	return ff(item)
}

// MarshalInit calls the wrapped function.
func (ff FormatFunc) MarshalInit(item interface{}) ([]byte, error) {
	return ff(item)
}

// MarshalItem calls the wrapped function.
func (ff FormatFunc) MarshalItem(item interface{}) ([]byte, error) {
	return ff(item)
}

// FrameItem simply adds a newline.
func (ff FormatFunc) FrameItem(buf []byte) ([]byte, error) {
	n := len(buf)
	frame := make([]byte, n+1)
	copy(frame, buf)
	frame[n] = '\n'
	return frame, nil
}
