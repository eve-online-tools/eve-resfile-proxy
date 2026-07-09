package fetch

type Semaphore struct {
	ch chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	if limit < 1 {
		limit = 1
	}
	return &Semaphore{ch: make(chan struct{}, limit)}
}

func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.ch
}
