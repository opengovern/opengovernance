package cost

import (
	"fmt"

	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/backend"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/query"
)

// State represents a collection of all the Resource costs (either prior or planned.) It is not tied to any specific
// cloud provider or IaC tool. Instead, it is a representation of a snapshot of cloud resources at a given point
// in time, with their associated costs.
type State struct {
	Resources map[string]Resource
}

// Errors that might be returned from NewState if either a product or a price are not found.
var (
	ErrProductNotFound = fmt.Errorf("product not found")
	ErrPriceNotFound   = fmt.Errorf("price not found")
)

// NewState returns a new State from a query.Resource slice by using the Backend to fetch the pricing data.
func NewState(backend backend.Backend, queries []query.Resource) (*State, error) {
	state := &State{Resources: make(map[string]Resource)}

	if len(queries) == 0 {
		return nil, fmt.Errorf("no query")
	}
	for _, res := range queries {
		// Mark the Resource as skipped if there are no valid Components.
		state.ensureResource(res.Address, res.Provider, res.Type, len(res.Components) == 0)

		for _, comp := range res.Components {
			price, err := backend.Products().Filter(comp.ProductFilter)
			if err != nil {
				state.addComponent(res.Address, comp.Name, Component{Error: err})
				continue
			}

			quantity := comp.MonthlyQuantity
			rate := NewMonthly(price.Price, "USD")

			if quantity.IsZero() {
				quantity = comp.HourlyQuantity
				rate = NewHourly(price.Price, "USD")
			}

			component := Component{
				Quantity: quantity,
				Unit:     comp.Unit,
				Rate:     rate,
				Details:  comp.Details,
				Usage:    comp.Usage,
			}

			state.addComponent(res.Address, comp.Name, component)
		}
	}

	return state, nil
}

// Cost returns the sum of the costs of every Resource included in this State.
// Error is returned if there is a mismatch in resource currencies.
func (s *State) Cost() (Cost, error) {
	var total Cost
	for name, re := range s.Resources {
		rCost, err := re.Cost()
		if err != nil {
			return Zero, fmt.Errorf("failed to get cost of resource %s: %w", name, err)
		}
		total, err = total.Add(rCost)
		if err != nil {
			return Zero, fmt.Errorf("failed to add cost of resource %s: %w", name, err)
		}
	}

	return total, nil
}

// ensureResource creates Resource at the given address if it doesn't already exist.
func (s *State) ensureResource(address, provider, typ string, skipped bool) {
	if _, ok := s.Resources[address]; !ok {
		res := Resource{
			Provider: provider,
			Type:     typ,
			Skipped:  skipped,
		}

		if !skipped {
			res.Components = make(map[string]Component)
		}

		s.Resources[address] = res
	}
}

// addComponent adds the Component with given label to the Resource at given address.
func (s *State) addComponent(resAddress, compLabel string, component Component) {
	s.Resources[resAddress].Components[compLabel] = component
}
