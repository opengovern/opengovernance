package compliance

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type GitParser struct {
	benchmarks []db.Benchmark
	controls   []db.Control
	queries    []db.Query
}

func (g *GitParser) ExtractQueries(queryPath string) error {
	return filepath.WalkDir(queryPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var query Query
			err = json.Unmarshal(content, &query)
			if err != nil {
				return err
			}

			g.queries = append(g.queries, db.Query{
				ID:             query.ID,
				QueryToExecute: query.QueryToExecute,
				Connector:      query.Connector,
				ListOfTables:   query.ListOfTables,
				Engine:         query.Engine,
				PrimaryTable:   query.PrimaryTable,
			})
		}

		return nil
	})
}

func (g *GitParser) ExtractControls(complianceControlsPath string) error {
	return filepath.WalkDir(complianceControlsPath, func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".json") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var control Control
			err = json.Unmarshal(content, &control)
			if err != nil {
				return err
			}
			tags := make([]db.ControlTag, 0, len(control.Tags))
			for tagKey, tagValue := range control.Tags {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   tagKey,
						Value: tagValue,
					},
					ControlID: control.ID,
				})
			}
			p := db.Control{
				ID:                 control.ID,
				Title:              control.Title,
				Description:        control.Description,
				Tags:               tags,
				Enabled:            true,
				Benchmarks:         nil,
				Severity:           types.ParseFindingSeverity(control.Severity),
				ManualVerification: control.ManualVerification,
				Managed:            control.Managed,
			}

			if control.Query != nil {
				g.queries = append(g.queries, db.Query{
					ID:             control.ID,
					QueryToExecute: control.Query.QueryToExecute,
					Connector:      control.Query.Connector,
					PrimaryTable:   control.Query.PrimaryTable,
					ListOfTables:   control.Query.ListOfTables,
					Engine:         control.Query.Engine,
				})
				p.QueryID = &control.ID
			}
			g.controls = append(g.controls, p)
		}
		return nil
	})
}

func (g *GitParser) ExtractBenchmarks(complianceBenchmarksPath string) error {
	var benchmarks []Benchmark
	err := filepath.WalkDir(complianceBenchmarksPath, func(path string, d fs.DirEntry, err error) error {
		if filepath.Base(path) == "children.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var objs []Benchmark
			err = json.Unmarshal(content, &objs)
			if err != nil {
				return err
			}
			benchmarks = append(benchmarks, objs...)
		}
		if filepath.Base(path) == "root.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var obj Benchmark
			err = json.Unmarshal(content, &obj)
			if err != nil {
				return err
			}
			benchmarks = append(benchmarks, obj)
		}
		return nil
	})

	if err != nil {
		return err
	}

	children := map[string][]string{}
	for _, o := range benchmarks {
		tags := make([]db.BenchmarkTag, 0, len(o.Tags))
		for tagKey, tagValue := range o.Tags {
			tags = append(tags, db.BenchmarkTag{
				Tag: model.Tag{
					Key:   tagKey,
					Value: tagValue,
				},
				BenchmarkID: o.ID,
			})
		}
		connector, _ := source.ParseType(o.Connector)

		b := db.Benchmark{
			ID:          o.ID,
			Title:       o.Title,
			DisplayCode: o.DisplayCode,
			Connector:   connector,
			Description: o.Description,
			Enabled:     o.Enabled,
			Managed:     o.Managed,
			AutoAssign:  o.AutoAssign,
			Baseline:    o.Baseline,
			Tags:        tags,
			Children:    nil,
			Controls:    nil,
		}
		for _, controls := range g.controls {
			if contains(o.Controls, controls.ID) {
				b.Controls = append(b.Controls, controls)
			}
		}
		if len(o.Controls) != len(b.Controls) {
			//fmt.Printf("could not find some controls, %d != %d", len(o.Controls), len(b.Controls))
		}
		g.benchmarks = append(g.benchmarks, b)
		children[o.ID] = o.Children
	}

	for idx, benchmark := range g.benchmarks {
		for _, childID := range children[benchmark.ID] {
			for _, child := range g.benchmarks {
				if child.ID == childID {
					benchmark.Children = append(benchmark.Children, child)
				}
			}
		}

		if len(children[benchmark.ID]) != len(benchmark.Children) {
			//fmt.Printf("could not find some benchmark children, %d != %d", len(children[benchmark.ID]), len(benchmark.Children))
		}
		g.benchmarks[idx] = benchmark
	}
	return nil
}

func (g *GitParser) CheckForDuplicate() error {
	visited := map[string]bool{}
	for _, b := range g.benchmarks {
		if _, ok := visited[b.ID]; !ok {
			visited[b.ID] = true
		} else {
			return fmt.Errorf("duplicate benchmark id: %s", b.ID)
		}
	}

	//ivisited := map[uint]bool{}
	//for _, b := range g.benchmarkTags {
	//	if _, ok := ivisited[b.ID]; !ok {
	//		ivisited[b.ID] = true
	//	} else {
	//		return fmt.Errorf("duplicate benchmark tag id: %d", b.ID)
	//	}
	//}

	//visited = map[string]bool{}
	//for _, b := range g.controls {
	//	if _, ok := visited[b.ID]; !ok {
	//		visited[b.ID] = true
	//	} else {
	//		return fmt.Errorf("duplicate control id: %s", b.ID)
	//	}
	//}

	//ivisited = map[uint]bool{}
	//for _, b := range g.controlTags {
	//	if _, ok := ivisited[b.ID]; !ok {
	//		ivisited[b.ID] = true
	//	} else {
	//		return fmt.Errorf("duplicate control tag id: %s", b.ID)
	//	}
	//}

	//visited = map[string]bool{}
	//for _, b := range g.queries {
	//	if _, ok := visited[b.ID]; !ok {
	//		visited[b.ID] = true
	//	} else {
	//		return fmt.Errorf("duplicate query id: %s", b.ID)
	//	}
	//}

	return nil
}

func (g *GitParser) ExtractCompliance(compliancePath string) error {
	if err := g.ExtractControls(path.Join(compliancePath, "controls")); err != nil {
		return err
	}
	if err := g.ExtractBenchmarks(path.Join(compliancePath, "benchmarks")); err != nil {
		return err
	}
	if err := g.CheckForDuplicate(); err != nil {
		return err
	}
	return nil
}

func contains[T uint | int | string](arr []T, ob T) bool {
	for _, o := range arr {
		if o == ob {
			return true
		}
	}
	return false
}
