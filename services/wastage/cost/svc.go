package cost

type Service struct {
	pennywiseBaseUrl string
}

func New(pennywiseBaseUrl string) *Service {
	return &Service{
		pennywiseBaseUrl: pennywiseBaseUrl,
	}
}
