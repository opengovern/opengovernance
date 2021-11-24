package describer

type Resource struct {
	ID          string
	Description interface{}
}

func (r Resource) UniqueID() string {
	return r.ID
}
