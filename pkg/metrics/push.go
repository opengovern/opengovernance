package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Push struct {
	Address string
	Job     string

	Pusher *push.Pusher
}

func (p *Push) AddMetric(c prometheus.Collector) {
	p.Pusher.Collector(c)
}

func (p *Push) Push() {
	p.Push()
}
