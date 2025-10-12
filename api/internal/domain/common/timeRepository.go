package common

import "time"

type TimeRepository interface {
	Now() time.Time
}
