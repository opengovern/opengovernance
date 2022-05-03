package compliance_report

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader         = "elasticsearch_index"
	ComplianceReportIndex = "compliance_report"
	AccountReportIndex    = "account_report"
)

type ReportType string

const (
	ReportTypeBenchmark = "benchmark"
	ReportTypeControl   = "control"
	ReportTypeResult    = "result"
)

type ReportResultObj struct {
	Result         Result   `json:"result"`
	ControlId      string   `json:"controlId"`
	ParentGroupIds []string `json:"parentGroupIDs"`
}

type ReportGroupObj struct {
	ID             string            `json:"id"` // benchmark id / control id
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Tags           map[string]string `json:"tags"`
	Summary        Summary           `json:"summary"`
	ChildGroupIds  []string          `json:"groupIDs"`
	ControlIds     []string          `json:"controlIDs"`
	ParentGroupIds []string          `json:"parentGroupIDs"`
	Level          int               `json:"level"`
}

type Report struct {
	Result      *ReportResultObj `json:"result,omitempty"`
	Group       *ReportGroupObj  `json:"group,omitempty"`
	Type        ReportType       `json:"type"`
	ReportJobId uint             `json:"reportJobID"`
	SourceID    uuid.UUID        `json:"sourceID"`
	Provider    utils.SourceType `json:"provider"`
	DescribedAt int64            `json:"describedAt"`
}

func (r Report) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	u, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(u.String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(ComplianceReportIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}

func (r Report) MessageID() string {
	return strconv.FormatInt(int64(r.ReportJobId), 10)
}

type AccountReport struct {
	SourceID             uuid.UUID        `json:"sourceID"`
	Provider             utils.SourceType `json:"provider"`
	BenchmarkID          string           `json:"benchmarkID"`
	ReportJobId          uint             `json:"reportJobID"`
	Summary              Summary          `json:"summary"`
	CreatedAt            int64            `json:"createdAt"`
	TotalResources       int              `json:"totalResources"`
	TotalCompliant       int              `json:"totalCompliant"`
	CompliancePercentage float64          `json:"compliancePercentage"`
}

func (r AccountReport) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	u, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(u.String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(AccountReportIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}

func (r AccountReport) MessageID() string {
	return r.SourceID.String()
}

type SummaryStatus struct {
	Alarm int `json:"alarm"`
	OK    int `json:"ok"`
	Info  int `json:"info"`
	Skip  int `json:"skip"`
	Error int `json:"error"`
}

type Summary struct {
	Status SummaryStatus `json:"status"`
}

type Dimension struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResultStatus string

const (
	ResultStatusAlarm = "alarm"
	ResultStatusInfo  = "info"
	ResultStatusOK    = "ok"
	ResultStatusSkip  = "skip"
	ResultStatusError = "error"
)

func (r ResultStatus) SeverityLevel() int {
	switch r {
	case ResultStatusOK:
		return 0
	case ResultStatusSkip:
		return 1
	case ResultStatusInfo:
		return 2
	case ResultStatusError:
		return 3
	case ResultStatusAlarm:
		return 4
	default:
		return -1
	}
}

type Result struct {
	Reason     string       `json:"reason"`
	Resource   string       `json:"resource"`
	Status     ResultStatus `json:"status" enums:"ok,info,skip,alarm,error"`
	Dimensions []Dimension  `json:"dimensions"`
}

type Control struct {
	Results     []Result          `json:"results"`
	ControlId   string            `json:"control_id"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Tags        map[string]string `json:"tags"`
	Title       string            `json:"title"`
}

type Group struct {
	GroupId     string            `json:"group_id"` // benchmark id / control id
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Summary     Summary           `json:"summary"`
	Groups      []Group           `json:"groups"`
	Controls    []Control         `json:"controls"`
}

type ReportQueryResponse struct {
	Hits ReportQueryHits `json:"hits"`
}
type ReportQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []ReportQueryHit  `json:"hits"`
}
type ReportQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  Report        `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

type AccountReportQueryResponse struct {
	Hits AccountReportQueryHits `json:"hits"`
}
type AccountReportQueryHits struct {
	Total keibi.SearchTotal       `json:"total"`
	Hits  []AccountReportQueryHit `json:"hits"`
}
type AccountReportQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  AccountReport `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

func ExtractNodes(root Group, provider utils.SourceType, tree []string, reportJobID uint, sourceID uuid.UUID, describedAt int64) []Report {
	var nodes []Report

	var controlIds, childGroupIds []string
	for _, control := range root.Controls {
		controlIds = append(controlIds, control.ControlId)
	}
	for _, group := range root.Groups {
		childGroupIds = append(childGroupIds, group.GroupId)
	}

	me := Report{
		Group: &ReportGroupObj{
			ID:             root.GroupId,
			Title:          root.Title,
			Description:    root.Description,
			Tags:           root.Tags,
			Summary:        root.Summary,
			ChildGroupIds:  childGroupIds,
			ControlIds:     controlIds,
			ParentGroupIds: tree,
			Level:          len(tree),
		},
		ReportJobId: reportJobID,
		Type:        ReportTypeBenchmark,
		SourceID:    sourceID,
		Provider:    provider,
		DescribedAt: describedAt,
	}
	nodes = append(nodes, me)

	newTree := make([]string, 0, len(tree)+1)
	newTree = append(newTree, tree...)
	newTree = append(newTree, root.GroupId)

	for _, control := range root.Controls {
		controlNode := Report{
			Group: &ReportGroupObj{
				ID:             control.ControlId,
				Title:          control.Title,
				Description:    control.Description,
				Tags:           control.Tags,
				Summary:        Summary{Status: SummaryStatus{}},
				ChildGroupIds:  nil,
				ControlIds:     nil,
				ParentGroupIds: newTree,
				Level:          len(tree),
			},
			ReportJobId: reportJobID,
			Type:        ReportTypeControl,
			SourceID:    sourceID,
			Provider:    provider,
			DescribedAt: describedAt,
		}
		nodes = append(nodes, controlNode)
	}

	for _, group := range root.Groups {
		newNodes := ExtractNodes(group, provider, newTree, reportJobID, sourceID, describedAt)
		nodes = append(nodes, newNodes...)
	}

	return nodes
}

func ExtractLeaves(root Group, provider utils.SourceType, tree []string, reportJobID uint, sourceID uuid.UUID, createdAt int64) []Report {
	var leaves []Report
	if root.Controls != nil {
		for _, control := range root.Controls {
			controlTree := make([]string, 0, len(tree)+1)
			controlTree = append(controlTree, tree...)
			controlTree = append(controlTree, control.ControlId)

			for _, result := range control.Results {
				leaves = append(leaves, Report{
					Result: &ReportResultObj{
						ControlId:      control.ControlId,
						Result:         result,
						ParentGroupIds: controlTree,
					},
					ReportJobId: reportJobID,
					Type:        ReportTypeResult,
					SourceID:    sourceID,
					Provider:    provider,
					DescribedAt: createdAt,
				})
			}
		}
		return leaves
	}

	newTree := make([]string, 0, len(tree)+1)
	newTree = append(newTree, tree...)
	newTree = append(newTree, root.GroupId)

	for _, group := range root.Groups {
		newLeaves := ExtractLeaves(group, provider, newTree, reportJobID, sourceID, createdAt)
		leaves = append(leaves, newLeaves...)
	}
	return leaves
}

func ParseReport(path string, reportJobID uint, sourceID uuid.UUID, describedAt int64, provider utils.SourceType) ([]Report, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root Group
	err = json.Unmarshal(content, &root)
	if err != nil {
		return nil, err
	}

	nodes := ExtractNodes(root, provider, nil, reportJobID, sourceID, describedAt)
	leaves := ExtractLeaves(root, provider, nil, reportJobID, sourceID, describedAt)
	return append(nodes, leaves...), nil
}

func QueryReports(sourceID uuid.UUID, jobIDs []int, types []ReportType, groupID *string, containsParentGroupId *string, size int, searchAfter []interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	var filters []interface{}
	var jobIDsStr []string
	for _, jobID := range jobIDs {
		jobIDsStr = append(jobIDsStr, strconv.Itoa(jobID))
	}

	var typesStr []string
	for _, t := range types {
		typesStr = append(typesStr, string(t))
	}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"reportJobID": jobIDsStr},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"type": typesStr},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"sourceID": {sourceID.String()}},
	})
	if groupID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"group.id": {*groupID}},
		})
	}
	if containsParentGroupId != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"result.parentGroupIDs": {*containsParentGroupId}},
		})
	}

	res["size"] = size
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	return res
}

func QueryReportsFrom(sourceID uuid.UUID, jobIDs []int, types []ReportType, groupID *string, containsParentGroupId *string, size, lastIdx int) map[string]interface{} {
	res := make(map[string]interface{})
	var filters []interface{}
	var jobIDsStr []string
	for _, jobID := range jobIDs {
		jobIDsStr = append(jobIDsStr, strconv.Itoa(jobID))
	}

	var typesStr []string
	for _, t := range types {
		typesStr = append(typesStr, string(t))
	}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"reportJobID": jobIDsStr},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"type": typesStr},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"sourceID": {sourceID.String()}},
	})
	if groupID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"group.id": {*groupID}},
		})
	}
	if containsParentGroupId != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"result.parentGroupIDs": {*containsParentGroupId}},
		})
	}

	res["size"] = size
	res["from"] = lastIdx

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	return res
}

func QueryTrend(sourceID uuid.UUID, benchmarkID string, createdAtFrom, createdAtTo int64, size int32, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"type": {"benchmark"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"sourceID": {sourceID.String()}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"group.id": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"describedAt": map[string]string{
				"gte": strconv.FormatInt(createdAtFrom, 10),
				"lte": strconv.FormatInt(createdAtTo, 10),
			},
		},
	})
	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

func QueryProviderResult(benchmarkID string, createdAt int64, order string, size int32, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"benchmarkID": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"createdAt": {createdAt}},
	})
	res["size"] = size
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	res["sort"] = []map[string]interface{}{
		{
			"compliancePercentage": order,
		},
		{
			"_id": "asc",
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

func QueryBenchmarks(provider *string, createdAt int64, level, size int32, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"type": {"benchmark"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"group.parentGroupIDs": {"root_result_group"}},
	})
	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"provider": {*provider}},
		})
	}
	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"describedAt": map[string]interface{}{
				// just to make sure we get the benchmarks at the time
				"gte": createdAt - 1*60*1000,
				"lte": createdAt + 1*60*1000,
			},
		},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"group.level": {level}},
	})
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}
	res["size"] = size
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}

	b, err := json.Marshal(res)
	return string(b), err
}
