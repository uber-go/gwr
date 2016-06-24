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

package expvar

import (
	"expvar"
	"fmt"
	"time"

	"github.com/uber-go/gwr/source"
)

// expvar.Do(func(kv expvar.KeyValue) {
// })

// expvar.Get(name string) expvar.Var

// Getter implements a get-able gwr data source around an expvar.
type Getter struct {
	expvar.KeyValue
	name string
}

// NewGetter creates a getter around an expvar, the gwr name is set to
// "/expvar/KEY".
func NewGetter(kv expvar.KeyValue) *Getter {
	return &Getter{
		KeyValue: kv,
		name:     fmt.Sprintf("/expvar/%s", kv.Key),
	}
}

// TODO: NewNamedGetter ?

// TODO: Add* conveniences

// Name returns a name like "/expvar/..." where "..." describes the expvar.
func (g *Getter) Name() string {
	return g.name
}

// Get returns the expvar's value.
func (g *Getter) Get() interface{} {
	if fn, ok := g.Value.(expvar.Func); ok {
		return fn()
	}
	return g.Value
}

// Poller  XXX why not only if changed
type Poller struct {
	Getter

	interval time.Duration
	ticker   *time.Ticker
	watcher  source.GenericDataWatcher
}

// NewPoller XXX
func NewPoller(kv expvar.KeyValue, interval time.Duration) *Poller {
	return &Poller{
		Getter: Getter{
			KeyValue: kv,
			name:     fmt.Sprintf("/expvar/%s/%s", kv.Key, interval),
		},
		interval: interval,
	}
}

// SetWatcher sets the reference to the watcher at source creation time.
func (p *Poller) SetWatcher(watcher source.GenericDataWatcher) {
	p.watcher = watcher
}

type timeValue struct {
	Time  time.Time   `json:"time"`
	Value interface{} `json:"value"`
}

func (tv timeValue) String() string {
	return fmt.Sprintf("%s %v", tv.Time, tv.Value)
}

// WatchInit is called to provide a new watcher with initial data.
func (p *Poller) WatchInit() interface{} {
	return timeValue{
		Time:  time.Now(),
		Value: p.Get(),
	}
}

// Activate is called when a watcher is added and there were none.
func (p *Poller) Activate() {
	if p.ticker != nil {
		return
	}
	p.ticker = time.NewTicker(p.interval)
	go func() {
		for t := range p.ticker.C {
			if !p.watcher.Active() {
				break
			}
			// TODO: if changed check
			if !p.watcher.HandleItem(timeValue{
				Time:  t,
				Value: p.Get(),
			}) {
				break
			}
		}
		p.ticker.Stop()
		p.ticker = nil
	}()
}
