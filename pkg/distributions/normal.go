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
	NormalSigma = 4
)

// Normal represents a normal (Gaussian) distribution (https://en.wikipedia.org/wiki/Normal_distribution).
type Normal struct {
	Impl distuv.Normal

	Number
	x float64
}

// NewNormal creates a new Normal distribution.
func NewNormal(lb int64, ub int64) *Normal {
	return &Normal{
		Impl: distuv.Normal{
			Mu:    float64(lb + ub/2), // Mean of the normal distribution
			Sigma: NormalSigma,        // Standard deviation of the normal distribution
		},
	}
}

// Next computes the value of the probability density function at x.
func (u *Normal) Next() float64 {
	n := u.Impl.Prob(u.x)

	u.x++

	return n
}
