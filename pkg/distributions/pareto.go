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

const (
	DefaultParetoScale = 1
	DefaultParetoShape = 0.1
)

// Pareto implements the Pareto (Type I) distribution
type Pareto struct {
	Impl distuv.Pareto

	Number
	x float64
}

// NewPareto creates a new Pareto distribution.
func NewPareto(scale float64, shape float64) *Pareto {
	return &Pareto{
		Impl: distuv.Pareto{
			Xm:    scale,
			Alpha: shape,
		},
	}
}

// Next computes the value of the probability density function at x.
func (u *Pareto) Next() float64 {
	n := u.Impl.Prob(u.x)

	u.x++

	return n
}
