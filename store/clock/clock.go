package clock

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct {
}

func (r *RealClock) Now() time.Time {
	return time.Now()
}

type UnRealClock struct {
	current int64
}

func (ur *UnRealClock) Now() time.Time {
	return time.Unix(0, ur.current)
}

func (ur *UnRealClock) AddSeconds(seconds int64) time.Time {
	ur.current += seconds * 1000000000
	return time.Unix(0, ur.current)
}

func (ur *UnRealClock) Add(nanoSeconds int64) time.Time {
	ur.current += nanoSeconds
	return time.Unix(0, ur.current)
}

func (ur *UnRealClock) ResetTime(nanoSeconds int64) {
	ur.current = nanoSeconds
}

func (ur *UnRealClock) ResetTimeSeconds(seconds int64) {
	ur.current = seconds * 1000000000
}
