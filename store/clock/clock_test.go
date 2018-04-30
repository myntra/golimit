package clock

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	clock := RealClock{}

	tr := clock.Now().UnixNano()
	ac := time.Now().UnixNano()
	assert.Equal(t, tr/1000000000, ac/1000000000)
}

func TestUnRealClock_Now(t *testing.T) {
	clock := UnRealClock{}
	assert.Equal(t, clock.Now().UnixNano(), int64(0))

	clock.AddSeconds(10)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000000000))
}

func TestUnRealClock_Add(t *testing.T) {
	clock := UnRealClock{}
	assert.Equal(t, clock.Now().UnixNano(), int64(0))

	clock.AddSeconds(10)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000000000))

	clock.Add(1000)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000001000))
}

func TestUnRealClock_AddSeconds(t *testing.T) {
	clock := UnRealClock{}
	assert.Equal(t, clock.Now().UnixNano(), int64(0))

	clock.AddSeconds(10)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000000000))

	clock.Add(1000)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000001000))
}

func TestUnRealClock_ResetTime(t *testing.T) {
	clock := UnRealClock{}
	assert.Equal(t, clock.Now().UnixNano(), int64(0))

	clock.AddSeconds(10)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000000000))

	clock.ResetTime(5000)
	assert.Equal(t, clock.Now().UnixNano(), int64(5000))
}

func TestUnRealClock_ResetTimeSeconds(t *testing.T) {
	clock := UnRealClock{}
	assert.Equal(t, clock.Now().UnixNano(), int64(0))

	clock.AddSeconds(10)
	assert.Equal(t, clock.Now().UnixNano(), int64(10000000000))

	clock.ResetTimeSeconds(5000)
	assert.Equal(t, clock.Now().UnixNano(), int64(5000000000000))
}
