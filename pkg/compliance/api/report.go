package api

import "github.com/kaytu-io/kaytu-engine/pkg/types"

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
	Reason     string                 `json:"reason"`
	Resource   string                 `json:"resource"`
	Status     types.ComplianceResult `json:"status" enums:"ok,info,skip,alarm,error"`
	Dimensions []Dimension            `json:"dimensions"`
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
