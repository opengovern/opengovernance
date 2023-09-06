package alerting

type RuleType struct {
	ID        uint   `json:"id"`
	EventType []byte `json:"event_type"`
	Scope     []byte `json:"scope"`
	Operator  string `json:"operator"`
	Value     int64  `json:"value"`
	ActionId  uint   `json:"action_id"`
}

type ActionType struct {
	ID      uint   `json:"id"`
	Method  string `json:"method"`
	Url     string `json:"url"`
	Headers []byte `json:"headers"`
	Body    string `json:"body"`
}
