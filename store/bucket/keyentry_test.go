package bucket

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/myntra/golimit/store/clock"
)

func TestEntry(t *testing.T) {

	//Expired entry check
	count := int32(1)
	_clock:= &clock.UnRealClock{}
	expiry := _clock.Now().UnixNano()
	e := NewEntry(count, expiry, _clock)
	assert.Equal(t, expiry, e.expires)
	assert.Equal(t, count, e.Count())

	assert.False(t, e.Expiry() < _clock.Now().UnixNano())
	_clock.Add(1)
	assert.True(t, e.Expiry() < _clock.Now().UnixNano())
	assert.Equal(t, true, e.Expired())
	assert.True(t, e.LastModified() < _clock.Now().UnixNano())

	e.ReInit(1, _clock.Now().UnixNano()+10000,_clock)

	e.Incr(2)

	count += 2

	assert.Equal(t, count, e.Count())

	e.Sync(5)
	count += 5
	assert.Equal(t, count, e.Count())

}
