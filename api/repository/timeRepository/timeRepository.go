package timeRepository

import "time"

type TimeRepository struct{}

func New() *TimeRepository {
	return &TimeRepository{}
}

func (t *TimeRepository) Now() time.Time {
	return time.Now()
}
