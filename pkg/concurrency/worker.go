package concurrency

import (
	"fmt"
	"sync"
)

type Work func() (interface{}, error)
type Result struct {
	Value interface{}
	Error error
}

type WorkPool struct {
	workChannel   chan Work
	resultChannel chan Result
	doneChannel   chan struct{}
	wg            sync.WaitGroup
	workerCount   int
	works         []Work
}

func NewWorkPool(workerCount int) *WorkPool {
	if workerCount < 1 {
		workerCount = 1
	}

	return &WorkPool{
		doneChannel: make(chan struct{}, workerCount),
		workerCount: workerCount,
		wg:          sync.WaitGroup{},
		works:       []Work{},
	}
}

func (w *WorkPool) AddJob(job Work) {
	w.works = append(w.works, job)
}

func (w *WorkPool) Run() []Result {
	w.workChannel = make(chan Work, len(w.works))
	w.resultChannel = make(chan Result, len(w.works))

	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker()
	}

	for _, work := range w.works {
		w.workChannel <- work
	}

	var r []Result
	for i := 0; i < len(w.works); i++ {
		result := <-w.resultChannel
		r = append(r, result)
	}

	for i := 0; i < w.workerCount; i++ {
		w.doneChannel <- struct{}{}
	}
	w.wg.Wait()
	close(w.doneChannel)
	close(w.workChannel)
	close(w.resultChannel)

	return r
}

func (w *WorkPool) worker() {
	for {
		select {
		case job := <-w.workChannel:
			v, err := func() (v interface{}, err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("paniced with %v", r)
						v = nil
					}
				}()
				return job()
			}()
			w.resultChannel <- Result{
				Value: v,
				Error: err,
			}
		case <-w.doneChannel:
			w.wg.Done()
			return
		}
	}
}
