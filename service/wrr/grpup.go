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

import (
	"fmt"
	"strconv"
	"strings"
)

type group struct {
	region string
	weight int
	IPs    []string
}

func (g group) String() string {
	return g.region + strconv.Itoa(g.weight)
}

// parseGroup parses "weight-eu-0-50" into group
func parseGroup(s string) (g *group, isweight bool, err error) {
	g = &group{}
	if !strings.HasPrefix(s, "weight") {
		return g, false, err
	}
	splits := strings.Split(s, "-")
	if len(splits) != 4 {
		return g, true, fmt.Errorf("invalid label: %s", s)
	}
	if splits[0] != "weight" {
		return g, true, fmt.Errorf("invalid label: %s", s)
	}
	g.region = splits[1]
	g.weight, err = strconv.Atoi(splits[3])
	return g, true, err
}