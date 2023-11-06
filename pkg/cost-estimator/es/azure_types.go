package es

import azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"

type MicrosoftVirtualMachineResponse struct {
	Took     int64 `json:"took"`
	TimedOut bool  `json:"timed_out"`
	Shards   struct {
		Total      int64 `json:"total"`
		Successful int64 `json:"successful"`
		Skipped    int64 `json:"skipped"`
		Failed     int64 `json:"failed"`
	}
	Hits struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		}
		MaxScore int64 `json:"max_score"`
		Hits     []struct {
			Index  string `json:"_index"`
			Type   string `json:"_type"`
			Id     string `json:"_id"`
			Score  int64  `json:"_score"`
			Source struct {
				Metadata     interface{}                            `json:"metadata"`
				SourceJobId  int64                                  `json:"source_job_id"`
				ResourceType string                                 `json:"resource_type"`
				CreatedAt    int64                                  `json:"created_at"`
				Description  azure.ComputeVirtualMachineDescription `json:"description"`
				ARN          string
				ID           string
				Name         string
				Account      string
				Region       string
				Partition    string
				Type         string
			} `json:"_source"`
		}
	}
}
