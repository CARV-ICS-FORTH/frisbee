// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package wait

import (
	"fmt"
	"sync"
)

/*
// ChannelMerge forwards the output of multiple channels into a single channel.
func ChannelMerge(cs ...<-chan interface{}) <-chan interface{} {
	var wg sync.WaitGroup

	wg.Add(len(cs))

	out := make(chan interface{})

	for _, c := range cs {
		go func(c <-chan interface{}) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

*/

// ChannelWaitForChildren blocks waiting for the returned children to be closed (by the caller) before
// closing the parent channel.
// The num is expected to be at least 1. If not, the functions panics.
func ChannelWaitForChildren(num int) (parent chan struct{}, children []chan struct{}) {
	if num == 0 {
		panic("num is expected to be greater than 0")
	}

	parent = make(chan struct{})

	children = make([]chan struct{}, num)

	var wg sync.WaitGroup

	wg.Add(len(children))

	for i := 0; i < num; i++ {
		children[i] = make(chan struct{})

		go func(c <-chan struct{}) {
			<-c
			wg.Done()
		}(children[i])
	}

	go func() {
		wg.Wait()
		close(parent)
	}()

	return parent, children
}

// ChannelWait blocks waiting for the received channels to be closed before closing the outgoing parent channel.
// The chans is expected to be at least 1. If not, the functions panics.
func ChannelWait(chans ...<-chan struct{}) <-chan struct{} {
	if len(chans) < 1 {
		panic(fmt.Sprintf("chans is expected to be greater than 1. Current: %d", len(chans)))
	}

	out := make(chan struct{})

	var wg sync.WaitGroup

	wg.Add(len(chans))

	for i := 0; i < len(chans); i++ {
		go func(c <-chan struct{}) {
			<-c
			wg.Done()
		}(chans[i])
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
