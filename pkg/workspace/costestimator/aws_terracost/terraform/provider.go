package terraform

import (
	"fmt"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws_terracost/region"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
)

// Provider is an implementation of the terraform.Provider, used to extract component queries from
// terraform resources.
type Provider struct {
	key    string
	region region.Code
}

// NewProvider returns a new Provider with the provided default region and a query key.
func NewProvider(key string, regionCode region.Code) (*Provider, error) {
	if !regionCode.Valid() {
		return nil, fmt.Errorf("invalid AWS region: %q", regionCode)
	}
	return &Provider{key: key, region: regionCode}, nil
}

// Name returns the Provider's common name.
func (p *Provider) Name() string { return p.key }

// ResourceComponents returns Component queries for a given terraform.Resource.
func (p *Provider) ResourceComponents(resourceType string, request any) ([]query.Component, error) {
	switch resourceType {
	case "aws_instance":
		vals, err := decodeInstanceValues(tfRes.Values)
		if err != nil {
			return nil, err
		}
		return p.newInstance(vals).Components(), nil
	case "aws_db_instance":
		vals, err := decodeDBInstanceValues(tfRes.Values)
		if err != nil {
			return nil, err
		}
		return p.newDBInstance(vals).Components(), nil
	case "aws_ebs_volume":
		vals, err := decodeVolumeValues(tfRes.Values)
		if err != nil {
			return nil, err
		}
		return p.newVolume(vals).Components(), nil
	case "aws_elb":
		// ELB Classic does not have any special configuration.
		vals := lbValues{LoadBalancerType: "classic"}
		return p.newLB(vals).Components(), nil

	case "aws_lb", "aws_alb":
		vals, err := decodeLBValues(tfRes.Values)
		if err != nil {
			return nil, err
		}
		return p.newLB(vals).Components(), nil

	default:
		return nil, nil
	}
}
