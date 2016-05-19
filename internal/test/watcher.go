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
