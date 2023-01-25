/*
Copyright 2021-2023 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package distributions

// Number is a common generator.
type Number struct {
	LastValue float64
}

// SetLastValue sets the last value generated.
func (n *Number) SetLastValue(value float64) {
	n.LastValue = value
}

// Last implements the Generator Last interface.
func (n *Number) Last() float64 {
	return n.LastValue
}
