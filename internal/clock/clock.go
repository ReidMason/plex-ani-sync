package clock

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func New() *RealClock {
	return &RealClock{}
}

func (c RealClock) Now() time.Time {
	return time.Now()
}

type MockClock struct {
	current time.Time
}

func NewMockClock(current time.Time) *MockClock {
	return &MockClock{current: current}
}

func (c MockClock) Now() time.Time {
	return c.current
}
