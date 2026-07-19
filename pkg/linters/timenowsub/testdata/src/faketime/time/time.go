package time

import stdtime "time"

type Time struct{}

func Now() Time {
	return Time{}
}

func (Time) Sub(Time) stdtime.Duration {
	return 0
}
