package kool

import "time"

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func MultiplyDuration[N Number](d time.Duration, n N) time.Duration {
	return time.Duration(int64(d) * int64(n))
}
