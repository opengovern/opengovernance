package compliance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gorm.io/gorm"
)

type GitParser struct {
	benchmarks    []db.Benchmark
	benchmarkTags []db.BenchmarkTag
	policies      []db.Policy
	policyTags    []db.PolicyTag
	queries       []db.Query
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
				Policies:       nil,
			})
		}

		return nil
	})
}

func (g *GitParser) ExtractBenchmarkTags(compliancePath string) error {
	return filepath.WalkDir(compliancePath, func(path string, d fs.DirEntry, err error) error {
		if filepath.Base(path) == "tags.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var objs []BenchmarkTag
			err = json.Unmarshal(content, &objs)
			if err != nil {
				return err
			}
			for _, o := range objs {
				g.benchmarkTags = append(g.benchmarkTags, db.BenchmarkTag{
					Model: gorm.Model{
						ID: o.ID,
					},
					Key:        o.Key,
					Value:      o.Value,
					Benchmarks: nil,
				})
			}
		}
		return nil
	})
}

func (g *GitParser) ExtractPolicyTags(compliancePath string) error {
	return filepath.WalkDir(compliancePath, func(path string, d fs.DirEntry, err error) error {
		if filepath.Base(path) == "policyTags.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var objs []PolicyTag
			err = json.Unmarshal(content, &objs)
			if err != nil {
				return err
			}
			for _, o := range objs {
				g.policyTags = append(g.policyTags, db.PolicyTag{
					Model: gorm.Model{
						ID: o.ID,
					},
					Key:      o.Key,
					Value:    o.Value,
					Policies: nil,
				})
			}
		}
		return nil
	})
}

func (g *GitParser) ExtractPolicies(compliancePath string) error {
	return filepath.WalkDir(compliancePath, func(path string, d fs.DirEntry, err error) error {
		if filepath.Base(path) == "policies.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var objs []Policy
			err = json.Unmarshal(content, &objs)
			if err != nil {
				return err
			}
			for _, o := range objs {
				p := db.Policy{
					ID:                 o.ID,
					Title:              o.Title,
					Description:        o.Description,
					Tags:               nil,
					DocumentURI:        o.DocumentURI,
					QueryID:            o.QueryID,
					Benchmarks:         nil,
					Severity:           o.Severity,
					ManualVerification: o.ManualVerification,
					Managed:            o.Managed,
				}
				for _, tag := range g.policyTags {
					if contains(o.Tags, tag.ID) {
						p.Tags = append(p.Tags, tag)
					}
				}
				if len(o.Tags) != len(p.Tags) {
					fmt.Printf("could not find some policy tags, %d != %d", len(o.Tags), len(p.Tags))
				}

				if p.QueryID != nil {
					found := false
					for idx, q := range g.queries {
						if q.ID == *p.QueryID {
							found = true
							q.Policies = append(q.Policies, p)
							g.queries[idx] = q
						}
					}
					if !found {
						fmt.Printf("could not find query with id %s", *p.QueryID)
					}
				}
				g.policies = append(g.policies, p)
			}
		}
		return nil
	})
}

func (g *GitParser) ExtractBenchmarks(compliancePath string) error {
	var benchmarks []Benchmark
	err := filepath.WalkDir(compliancePath, func(path string, d fs.DirEntry, err error) error {
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
		b := db.Benchmark{
			ID:          o.ID,
			Title:       o.Title,
			Description: o.Description,
			LogoURI:     o.LogoURI,
			Category:    o.Category,
			DocumentURI: o.DocumentURI,
			Enabled:     o.Enabled,
			Managed:     o.Managed,
			AutoAssign:  o.AutoAssign,
			Baseline:    o.Baseline,
			Tags:        nil,
			Children:    nil,
			Policies:    nil,
		}
		for _, tag := range g.benchmarkTags {
			if contains(o.Tags, tag.ID) {
				b.Tags = append(b.Tags, tag)
			}
		}
		if len(o.Tags) != len(b.Tags) {
			fmt.Printf("could not find some benchmark tags, %d != %d", len(o.Tags), len(b.Tags))
		}
		for _, policy := range g.policies {
			if contains(o.Policies, policy.ID) {
				b.Policies = append(b.Policies, policy)
			}
		}
		if len(o.Policies) != len(b.Policies) {
			fmt.Printf("could not find some policies, %d != %d", len(o.Policies), len(b.Policies))
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
			fmt.Printf("could not find some benchmark children, %d != %d", len(children[benchmark.ID]), len(benchmark.Children))
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

	ivisited := map[uint]bool{}
	for _, b := range g.benchmarkTags {
		if _, ok := ivisited[b.ID]; !ok {
			ivisited[b.ID] = true
		} else {
			return fmt.Errorf("duplicate benchmark tag id: %d", b.ID)
		}
	}

	visited = map[string]bool{}
	for _, b := range g.policies {
		if _, ok := visited[b.ID]; !ok {
			visited[b.ID] = true
		} else {
			return fmt.Errorf("duplicate policy id: %s", b.ID)
		}
	}

	ivisited = map[uint]bool{}
	for _, b := range g.policyTags {
		if _, ok := ivisited[b.ID]; !ok {
			ivisited[b.ID] = true
		} else {
			return fmt.Errorf("duplicate policy tag id: %s", b.ID)
		}
	}

	visited = map[string]bool{}
	for _, b := range g.queries {
		if _, ok := visited[b.ID]; !ok {
			visited[b.ID] = true
		} else {
			return fmt.Errorf("duplicate query id: %s", b.ID)
		}
	}

	return nil
}

func (g *GitParser) ExtractCompliance(compliancePath string) error {
	if err := g.ExtractPolicyTags(compliancePath); err != nil {
		return err
	}
	if err := g.ExtractBenchmarkTags(compliancePath); err != nil {
		return err
	}
	if err := g.ExtractPolicies(compliancePath); err != nil {
		return err
	}
	if err := g.ExtractBenchmarks(compliancePath); err != nil {
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
