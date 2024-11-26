package compliance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/jackc/pgtype"
	"github.com/opengovern/og-util/pkg/model"
	"github.com/opengovern/opencomply/jobs/post-install-job/job/git"
	"github.com/opengovern/opencomply/pkg/types"
	"github.com/opengovern/opencomply/services/compliance/db"
	"github.com/opengovern/opencomply/services/metadata/models"
	"go.uber.org/zap"
)

type GitParser struct {
	logger          *zap.Logger
	benchmarks      []db.Benchmark
	controls        []db.Control
	queries         []db.Query
	queryParams     []models.QueryParameter
	queryViews      []models.QueryView
	controlsQueries map[string]db.Query
	Comparison      *git.ComparisonResultGrouped
}

func populateMdMapFromPath(path string) (map[string]string, error) {
	result := make(map[string]string)
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		id := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".md"))
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[id] = string(content)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *GitParser) ExtractControls(complianceControlsPath string, controlEnrichmentBasePath string) error {
	manualRemediationMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "remediation", "manual"))
	if err != nil {
		g.logger.Warn("failed to load manual remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded manual remediation", zap.Int("count", len(manualRemediationMap)))
	}

	cliRemediationMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "remediation", "cli"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	guardrailRemediationMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "remediation", "guardrail"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	programmaticRemediationMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "remediation", "programmatic"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	noncomplianceCostMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "noncompliance-cost"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	usefulnessExampleMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "usefulness-example"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	explanationMap, err := populateMdMapFromPath(path.Join(controlEnrichmentBasePath, "tags", "explanation"))
	if err != nil {
		g.logger.Warn("failed to load cli remediation", zap.Error(err))
	} else {
		// g.logger.Info("loaded cli remediation", zap.Int("count", len(cliRemediationMap)))
	}

	// g.logger.Info("extracting controls", zap.Int("manualRemediationMap", len(manualRemediationMap)),
	// 	zap.Int("cliRemediationMap", len(cliRemediationMap)), zap.Int("guardrailRemediationMap", len(guardrailRemediationMap)),
	// 	zap.Int("programmaticRemediationMap", len(programmaticRemediationMap)), zap.Int("noncomplianceCostMap", len(noncomplianceCostMap)),
	// 	zap.Int("usefulnessExampleMap", len(usefulnessExampleMap)), zap.Int("explanationMap", len(explanationMap)))

	return filepath.WalkDir(complianceControlsPath, func(path string, d fs.DirEntry, err error) error {
		//if g.Comparison != nil {
		//	_, modified := g.Comparison.ModifiedFiles[path]
		//	_, created := g.Comparison.CreatedFiles[path]
		//
		//	if !modified && !created {
		//		return nil
		//	}
		//}
		if strings.HasSuffix(path, ".yaml") {
			content, err := os.ReadFile(path)
			if err != nil {
				g.logger.Error("failed to read control", zap.String("path", path), zap.Error(err))
				return err
			}

			var control Control
			err = yaml.Unmarshal(content, &control)
			if err != nil {
				g.logger.Error("failed to unmarshal control", zap.String("path", path), zap.Error(err))
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
			if v, ok := manualRemediationMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-manual-remediation",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := cliRemediationMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-cli-remediation",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := guardrailRemediationMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-guardrail-remediation",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := programmaticRemediationMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-programmatic-remediation",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := noncomplianceCostMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-noncompliance-cost",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := explanationMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-explanation",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if v, ok := usefulnessExampleMap[strings.ToLower(control.ID)]; ok {
				tags = append(tags, db.ControlTag{
					Tag: model.Tag{
						Key:   "x-opengovernance-usefulness-example",
						Value: []string{v},
					},
					ControlID: control.ID,
				})
			}
			if control.Severity == "" {
				control.Severity = "low"
			}

			p := db.Control{
				ID:                 control.ID,
				Title:              control.Title,
				Description:        control.Description,
				Tags:               tags,
				IntegrationType:    control.IntegrationType,
				Enabled:            true,
				Benchmarks:         nil,
				Severity:           types.ParseComplianceResultSeverity(control.Severity),
				ManualVerification: control.ManualVerification,
				Managed:            control.Managed,
			}

			if control.Query != nil {
				q := db.Query{
					ID:              control.ID,
					QueryToExecute:  control.Query.QueryToExecute,
					IntegrationType: control.IntegrationType,
					PrimaryTable:    control.Query.PrimaryTable,
					ListOfTables:    control.Query.ListOfTables,
					Engine:          control.Query.Engine,
					Global:          control.Query.Global,
				}
				g.controlsQueries[control.ID] = q
				for _, parameter := range control.Query.Parameters {
					q.Parameters = append(q.Parameters, db.QueryParameter{
						QueryID:  control.ID,
						Key:      parameter.Key,
						Required: parameter.Required,
					})

					if parameter.DefaultValue != nil {
						g.queryParams = append(g.queryParams, models.QueryParameter{
							Key:   parameter.Key,
							Value: *parameter.DefaultValue,
						})
					}
				}
				g.queries = append(g.queries, q)
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
		if !strings.HasSuffix(filepath.Base(path), ".yaml") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			g.logger.Error("failed to read benchmark", zap.String("path", path), zap.Error(err))
			return err
		}

		var obj Benchmark
		err = yaml.Unmarshal(content, &obj)
		if err != nil {
			g.logger.Error("failed to unmarshal benchmark", zap.String("path", path), zap.Error(err))
			return err
		}
		benchmarks = append(benchmarks, obj)

		return nil
	})

	if err != nil {
		return err
	}
	// g.logger.Info("Extracted benchmarks 1", zap.Int("count", len(benchmarks)))

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

		autoAssign := true
		if o.AutoAssign != nil {
			autoAssign = *o.AutoAssign
		}

		b := db.Benchmark{
			ID:                o.ID,
			Title:             o.Title,
			DisplayCode:       o.SectionCode,
			Description:       o.Description,
			AutoAssign:        autoAssign,
			TracksDriftEvents: o.TracksDriftEvents,
			Tags:              tags,
			Children:          nil,
			Controls:          nil,
		}
		metadataJsonb := pgtype.JSONB{}
		err = metadataJsonb.Set([]byte(""))
		if err != nil {
			return err
		}
		b.Metadata = metadataJsonb

		for _, controls := range g.controls {
			if contains(o.Controls, controls.ID) {
				b.Controls = append(b.Controls, controls)
			}
		}

		integrationTypes := make(map[string]bool)
		for _, c := range b.Controls {
			for _, it := range c.IntegrationType {
				integrationTypes[it] = true
			}
		}
		var integrationTypesList []string
		for k, _ := range integrationTypes {
			integrationTypesList = append(integrationTypesList, k)
		}
		b.IntegrationType = integrationTypesList

		g.benchmarks = append(g.benchmarks, b)
		children[o.ID] = o.Children
	}
	// g.logger.Info("Extracted benchmarks 2", zap.Int("count", len(g.benchmarks)))

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
	// g.logger.Info("Extracted benchmarks 3", zap.Int("count", len(g.benchmarks)))

	g.benchmarks, _ = fillBenchmarksIntegrationTypes(g.benchmarks)
	g.logger.Info("Extracted benchmarks 4", zap.Int("count", len(g.benchmarks)))

	return nil
}

func fillBenchmarksIntegrationTypes(benchmarks []db.Benchmark) ([]db.Benchmark, []string) {
	var integrationTypes []string
	integrationTypesMap := make(map[string]bool)

	for idx, benchmark := range benchmarks {
		if benchmark.IntegrationType == nil {
			benchmark.Children, benchmark.IntegrationType = fillBenchmarksIntegrationTypes(benchmark.Children)
			benchmarks[idx] = benchmark
		}
		for _, c := range benchmark.IntegrationType {
			if _, ok := integrationTypesMap[c]; !ok {
				integrationTypes = append(integrationTypes, c)
				integrationTypesMap[c] = true
			}
		}
	}
	return benchmarks, integrationTypes
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

func (g GitParser) ExtractBenchmarksMetadata() error {
	for i, b := range g.benchmarks {
		benchmarkControlsCache := make(map[string]BenchmarkControlsCache)
		controlsMap, err := getControlsUnderBenchmark(b, benchmarkControlsCache)
		if err != nil {
			return err
		}
		benchmarkTablesCache := make(map[string]BenchmarkTablesCache)
		primaryTablesMap, listOfTablesMap, err := g.getTablesUnderBenchmark(b, benchmarkTablesCache)
		if err != nil {
			return err
		}
		var listOfTables, primaryTables, controls []string
		for k, _ := range controlsMap {
			controls = append(controls, k)
		}
		for k, _ := range primaryTablesMap {
			primaryTables = append(primaryTables, k)
		}
		for k, _ := range listOfTablesMap {
			listOfTables = append(listOfTables, k)
		}
		metadata := db.BenchmarkMetadata{
			Controls:      controls,
			PrimaryTables: primaryTables,
			ListOfTables:  listOfTables,
		}
		metadataJson, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		metadataJsonb := pgtype.JSONB{}
		err = metadataJsonb.Set(metadataJson)
		if err != nil {
			return err
		}
		g.benchmarks[i].Metadata = metadataJsonb
	}
	return nil
}

func (g *GitParser) ExtractCompliance(compliancePath string, controlEnrichmentBasePath string) error {
	if err := g.ExtractControls(path.Join(compliancePath, "controls"), controlEnrichmentBasePath); err != nil {
		return err
	}
	if err := g.ExtractBenchmarks(path.Join(compliancePath, "frameworks")); err != nil {
		return err
	}
	if err := g.CheckForDuplicate(); err != nil {
		return err
	}

	if err := g.ExtractBenchmarksMetadata(); err != nil {
		return err
	}
	return nil
}

func (g *GitParser) ExtractQueryViews(viewsPath string) error {
	return filepath.WalkDir(viewsPath, func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			g.logger.Error("failed to read query view", zap.String("path", path), zap.Error(err))
			return err
		}

		var obj QueryView
		err = yaml.Unmarshal(content, &obj)
		if err != nil {
			g.logger.Error("failed to unmarshal query view", zap.String("path", path), zap.Error(err))
			return err
		}

		g.queryViews = append(g.queryViews, models.QueryView{
			ID:           obj.ID,
			Query:        obj.Query,
			Dependencies: obj.Dependencies,
		})

		return nil
	})
}

func contains[T uint | int | string](arr []T, ob T) bool {
	for _, o := range arr {
		if o == ob {
			return true
		}
	}
	return false
}
