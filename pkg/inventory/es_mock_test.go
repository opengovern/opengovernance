package inventory

import (
	"context"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ElasticClientMock struct {
}

func (e ElasticClientMock) Search(ctx context.Context, index string, query string, response interface{}) error {
	if r, ok := response.(*SummaryQueryResponse); ok {
		*r = SummaryQueryResponse{
			Hits: SummaryQueryHits{
				Hits: []SummaryQueryHit{
					{
						Source: describe.KafkaLookupResource{
							ResourceID:    "resource-1",
							Name:          "1-name",
							SourceType:    "aws",
							ResourceType:  "AWS::EC2::Instance",
							ResourceGroup: "",
							Location:      "us-east1",
							SourceID:      "11111",
							ResourceJobID: 0,
							SourceJobID:   0,
						},
					},{
						Source: describe.KafkaLookupResource{
							ResourceID:    "resource-2",
							Name:          "2-name",
							SourceType:    "aws",
							ResourceType:  "AWS::EC2::Instance",
							ResourceGroup: "",
							Location:      "us-east2",
							SourceID:      "11112",
							ResourceJobID: 0,
							SourceJobID:   0,
						},
					},{
						Source: describe.KafkaLookupResource{
							ResourceID:    "resource-3",
							Name:          "3-name",
							SourceType:    "azure",
							ResourceType:  "Microsoft.Network/virtualNetworks",
							ResourceGroup: "ahkasdbh",
							Location:      "us-east1",
							SourceID:      "aaaaaa",
							ResourceJobID: 0,
							SourceJobID:   0,
						},
					},{
						Source: describe.KafkaLookupResource{
							ResourceID:    "resource-4",
							Name:          "4-name",
							SourceType:    "azure",
							ResourceType:  "Microsoft.Network/virtualNetworks",
							ResourceGroup: "ahkasdbh",
							Location:      "us-east2",
							SourceID:      "aaaaab",
							ResourceJobID: 0,
							SourceJobID:   0,
						},
					},
				},
			},
		}
	}
	return nil
}
func (e ElasticClientMock) NewEC2RegionPaginator(filters []keibi.BoolFilter, limit *int64) (keibi.EC2RegionPaginator, error) {
	return keibi.EC2RegionPaginator{}, nil
}
func (e ElasticClientMock) NewLocationPaginator(filters []keibi.BoolFilter, limit *int64) (keibi.LocationPaginator, error) {
	return keibi.LocationPaginator{}, nil
}
