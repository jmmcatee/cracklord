package queue

type Stats struct {
	jobsCount int64
}

func (s *Stats) IncJob() {
	s.jobsCount++
}

func (s *Stats) JobCount() int64 {
	return s.jobsCount
}
