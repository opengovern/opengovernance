package describer

type Resource struct {
	ID          string
	Description interface{}

	Subscription string
	Type         string
}

func (r Resource) UniqueID() string {
	return r.ID
}
