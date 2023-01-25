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

import (
	"gonum.org/v1/gonum/stat/distuv"
)

// Uniform represents a continuous uniform distribution (https://en.wikipedia.org/wiki/Uniform_distribution_%28continuous%29).
type Uniform struct {
	Impl distuv.Uniform

	Number
	x float64
}

// NewUniform creates a new Uniform distribution.
func NewUniform(lb int64, ub int64) *Uniform {
	return &Uniform{
		Impl: distuv.Uniform{
			Min: float64(lb),
			Max: float64(ub),
		},
		Number: Number{},
		x:      0,
	}
}

// Next computes the value of the probability density function at x.
func (u *Uniform) Next() float64 {
	n := u.Impl.Prob(u.x)

	u.x++

	return n
}
