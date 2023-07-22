package api

type RunQueryRequest struct {
	Page  Page                 `json:"page" validate:"required"`
	Query *string              `json:"query"`
	Sorts []SmartQuerySortItem `json:"sorts"`
}

type RunQueryResponse struct {
	Title    string   `json:"title"`    // Query Title
	Query    string   `json:"query"`    // Query
	RowCount int      `json:"rowCount"` // Number of rows in result
	Headers  []string `json:"headers"`  // Column names
	Result   [][]any  `json:"result"`   // Result of query. in order to access a specific cell please use Result[Row][Column]
}
