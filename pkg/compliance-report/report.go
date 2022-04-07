package compliance_report

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader         = "elasticsearch_index"
	ComplianceReportIndex = "compliance_report"
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
}

type Report struct {
	Result      *ReportResultObj `json:"result,omitempty"`
	Group       *ReportGroupObj  `json:"group,omitempty"`
	Type        ReportType       `json:"type"`
	ReportJobId uint             `json:"reportJobID"`
	SourceID    uuid.UUID        `json:"sourceID"`
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

type Result struct {
	Reason     string      `json:"reason"`
	Resource   string      `json:"resource"`
	Status     string      `json:"status"`
	Dimensions []Dimension `json:"dimensions"`
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

func ExtractNodes(root Group, tree []string, reportJobID uint, sourceID uuid.UUID) []Report {
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
		},
		ReportJobId: reportJobID,
		Type:        ReportTypeBenchmark,
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
			},
			ReportJobId: reportJobID,
			Type:        ReportTypeControl,
			SourceID:    sourceID,
		}
		nodes = append(nodes, controlNode)
	}

	for _, group := range root.Groups {
		newNodes := ExtractNodes(group, newTree, reportJobID, sourceID)
		nodes = append(nodes, newNodes...)
	}

	return nodes
}

func ExtractLeaves(root Group, tree []string, reportJobID uint, sourceID uuid.UUID) []Report {
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
				})
			}
		}
		return leaves
	}

	newTree := make([]string, 0, len(tree)+1)
	newTree = append(newTree, tree...)
	newTree = append(newTree, root.GroupId)

	for _, group := range root.Groups {
		newLeaves := ExtractLeaves(group, newTree, reportJobID, sourceID)
		leaves = append(leaves, newLeaves...)
	}
	return leaves
}

func ParseReport(path string, reportJobID uint, sourceID uuid.UUID) ([]Report, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root Group
	err = json.Unmarshal(content, &root)
	if err != nil {
		return nil, err
	}

	nodes := ExtractNodes(root, nil, reportJobID, sourceID)
	leaves := ExtractLeaves(root, nil, reportJobID, sourceID)
	return append(nodes, leaves...), nil
}

func QueryReports(sourceID uuid.UUID, jobIDs []int, type_ ReportType, groupID *string, size, lastIdx int) map[string]interface{} {
	res := make(map[string]interface{})
	var filters []interface{}
	var jobIDsStr []string
	for _, jobID := range jobIDs {
		jobIDsStr = append(jobIDsStr, strconv.Itoa(jobID))
	}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"reportJobID": jobIDsStr},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"type": {string(type_)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"sourceID": {sourceID.String()}},
	})
	if groupID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"group.id": {*groupID}},
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
