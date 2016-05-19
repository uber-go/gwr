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

package test

import "fmt"

type element struct {
	Item  interface{}
	Items []interface{}
}

// Watcher implements a source.GenericDataWatcher that appends to a slice for
// testing purposes.
type Watcher struct {
	Q []element
}

// NewWatcher creates a new test watcher.
func NewWatcher() *Watcher {
	return &Watcher{}
}

// Active always returns true.
func (wat *Watcher) Active() bool {
	return true
}

// HandleItem appends the item to the Q and always returns true.
func (wat *Watcher) HandleItem(item interface{}) bool {
	wat.Q = append(wat.Q, element{Item: item})
	return true
}

// HandleItems appends the items to the Q and always returns true.
func (wat *Watcher) HandleItems(items []interface{}) bool {
	wat.Q = append(wat.Q, element{Items: items})
	return true
}

// AllItems returns a list containing all of the items in the Q flattened out.
func (wat *Watcher) AllItems() (items []interface{}) {
	for _, el := range wat.Q {
		if el.Item != nil {
			items = append(items, el.Item)
		} else {
			items = append(items, el.Items...)
		}
	}
	return
}

// AllStrings retruns a list of strings called by either .String-ing each item
// in AllItems, or fmting it with "%#v".
func (wat *Watcher) AllStrings() (strs []string) {
	each := func(items ...interface{}) {
		for _, item := range items {
			if strer, ok := item.(fmt.Stringer); ok {
				strs = append(strs, strer.String())
			} else {
				strs = append(strs, fmt.Sprintf("%#v", item))
			}
		}
	}
	for _, el := range wat.Q {
		if el.Item != nil {
			each(el.Item)
		} else {
			each(el.Items...)
		}
	}
	return
}
