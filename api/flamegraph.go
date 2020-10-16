// Copyright 2020 The conprof Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"strings"

	"github.com/conprof/conprof/internal/pprof/graph"
	"github.com/conprof/conprof/internal/pprof/measurement"
	"github.com/conprof/conprof/internal/pprof/report"
	"github.com/google/pprof/profile"
)

type treeNode struct {
	Name      string      `json:"n"`
	FullName  string      `json:"f"`
	Cum       int64       `json:"v"`
	CumFormat string      `json:"l"`
	Percent   string      `json:"p"`
	Children  []*treeNode `json:"c"`
}

// Largely copied from https://github.com/google/pprof/blob/master/internal/driver/flamegraph.go
func generateFlamegraphReport(p *profile.Profile, sampleIndex string) (*treeNode, error) {
	numLabelUnits, _ := p.NumLabelUnits()
	err := p.Aggregate(false, true, true, true, false)
	if err != nil {
		return nil, err
	}

	value, meanDiv, sample, err := sampleFormat(p, sampleIndex, false)
	if err != nil {
		return nil, err
	}

	stype := sample.Type

	rep := report.New(p, &report.Options{
		OutputFormat:  report.Text,
		OutputUnit:    "minimum",
		Ratio:         1,
		NumLabelUnits: numLabelUnits,

		SampleValue:       value,
		SampleMeanDivisor: meanDiv,
		SampleType:        stype,
		SampleUnit:        sample.Unit,

		NodeCount:    80,
		NodeFraction: 0.005,
		EdgeFraction: 0.001,
	})

	g, config := report.GetDOT(rep)
	var nodes []*treeNode
	nroots := 0
	rootValue := int64(0)
	nodeMap := map[*graph.Node]*treeNode{}
	// Make all nodes and the map, collect the roots.
	for _, n := range g.Nodes {
		v := n.CumValue()
		fullName := n.Info.PrintableName()
		node := &treeNode{
			Name:      graph.ShortenFunctionName(fullName),
			FullName:  fullName,
			Cum:       v,
			CumFormat: config.FormatValue(v),
			Percent:   strings.TrimSpace(measurement.Percentage(v, config.Total)),
		}
		nodes = append(nodes, node)
		if len(n.In) == 0 {
			nodes[nroots], nodes[len(nodes)-1] = nodes[len(nodes)-1], nodes[nroots]
			nroots++
			rootValue += v
		}
		nodeMap[n] = node
	}
	// Populate the child links.
	for _, n := range g.Nodes {
		node := nodeMap[n]
		for child := range n.Out {
			node.Children = append(node.Children, nodeMap[child])
		}
	}

	return &treeNode{
		Name:      "root",
		FullName:  "root",
		Cum:       rootValue,
		CumFormat: config.FormatValue(rootValue),
		Percent:   strings.TrimSpace(measurement.Percentage(rootValue, config.Total)),
		Children:  nodes[0:nroots],
	}, nil
}
