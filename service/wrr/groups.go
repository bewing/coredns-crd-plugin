package wrr

/*
Copyright 2022 The k8gb Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Generated by GoLic, for more details see: https://github.com/AbsaOSS/golic
*/

import "sort"

type groups []*group

// parseGroups create slice of groups in order as they are defined in Endpoint
func parseGroups(labels map[string]string) (g groups, err error) {
	filter := make(map[string]*group, 0)
	for k, v := range labels {
		pg, weight, err := parseGroup(k)
		if !weight {
			continue
		}
		if err != nil {
			return g, err
		}
		if filter[pg.String()] == nil {
			filter[pg.String()] = pg
			g = append(g, pg)
		}
		filter[pg.String()].IPs = append(filter[pg.String()].IPs, v)
	}
	// labels argument is map, so the groups has random order compared to immutable
	sort.Slice(g, func(i, j int) bool {
		return g[i].String() < g[j].String()
	})

	return g, err
}

func (g groups) pdf() (pdf []uint32) {
	for _, v := range g {
		pdf = append(pdf, uint32(v.weight))
	}
	return pdf
}

func (g *groups) shuffle(vec []int) {
	var gg []*group
	for _, v := range vec {
		gg = append(gg, (*g)[v])
	}
	*g = gg
}

// asSlice converts groups to array of IP address
// Function respects order of groups
func (g groups) asSlice() (arr []string) {
	for _, v := range g {
		arr = append(arr, v.IPs...)
	}
	return arr
}

func (g groups) hasWeights() bool {
	return len(g) != 0
}

// equalIPs checks whether slice a and b contain the same elements in unsorted slices.
// A nil argument is equivalent to an empty slice.
func (g *groups) equalIPs(ips []string) bool {
	gs := g.asSlice()
	if len(gs) != len(ips) {
		return false
	}
	x := make([]string, len(ips))
	y := make([]string, len(gs))
	copy(x, ips)
	copy(y, gs)
	sort.Strings(x)
	sort.Strings(y)
	for i, v := range x {
		if v != y[i] {
			return false
		}
	}
	return true
}