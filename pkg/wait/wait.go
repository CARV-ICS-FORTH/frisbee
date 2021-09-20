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
	"sync"
)

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
