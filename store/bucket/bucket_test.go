package bucket

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/myntra/golimit/store/clock"
)

func TestBucket(t *testing.T) {
	_clock:= &clock.UnRealClock{}

	b := NewKeyBucket(_clock)
	blocked, expiry, justbreached := b.Incr("one", 1, 10, 1)

	assert.Equal(t,expiry ,_clock.Now().UnixNano()+1000000000)

	for i := 0; i < 9 && expiry > _clock.Now().UnixNano(); i++ {
		blocked, expiry, justbreached = b.Incr("one", 1, 10, 1)
		assert.False(t, blocked)
		assert.False(t, justbreached)
	}

	blocked, expiry, justbreached = b.Incr("one", 1, 10, 1)
	assert.True(t, blocked)
	assert.True(t, justbreached)

	_clock.Add(1000000001)

	blocked, expiry, justbreached = b.Incr("one", 1, 10, 1)
	assert.False(t, blocked)
	assert.False(t, justbreached)

	b.Sync("one", 5, expiry)
	e := b.GetEntry("one")
	assert.Equal(t, int32(6), e.Count())

	b.Sync("two", 5, expiry)
	e = b.GetEntry("two")
	assert.Equal(t, int32(5), e.Count())

	_clock.AddSeconds(1)

	b.Sync("two", 5, expiry)
	e = b.GetEntry("two")
	assert.Equal(t, int32(5), e.Count())

	b.Incr("three", 1, 10, 0)
	_clock.Add(1)
	e = b.GetEntry("three")
	assert.Equal(t, int32(1), e.Count())
	assert.True(t, e.Expired())

	b.Incr("three", 1, 10, 1)
	_clock.Add(1)
	e = b.GetEntry("three")
	assert.Equal(t, int32(1), e.Count())
	assert.False(t, e.Expired())
	_clock.Add(1000000000)
	e = b.GetEntry("three")
	assert.Equal(t, int32(1), e.Count())
	assert.True(t, e.Expired())


	b.Sync("three", 2, _clock.Now().UnixNano()+1000000000)
	_clock.Add(1000)
	e = b.GetEntry("three")
	assert.Equal(t, int32(2), e.Count())
	assert.False(t, e.Expired())


}


func TestGenExpiry(t *testing.T) {
	tm:=int64(0000000000)
	assert.Equal(t,int64(5000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(1000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))

	tm=int64(1000000000)
	assert.Equal(t,int64(5000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(2000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))

	tm=int64(2000000000)
	assert.Equal(t,int64(5000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(3000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))

	tm=int64(3000000000)
	assert.Equal(t,int64(5000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(4000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))

	tm=int64(5000000000)
	assert.Equal(t,int64(10000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(6000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))


	tm=int64(5999000000)
	assert.Equal(t,int64(10000000000),GenExpiry(tm,5))
	assert.Equal(t,int64(6000000000),GenExpiry(tm,1))
	assert.Equal(t,int64(10000000000),GenExpiry(tm,10))
}

func TestKeyBucket_Lookup(t *testing.T) {
	_clock:= &clock.UnRealClock{}
	b := NewKeyBucket(_clock)
	b.Incr("key",1,1,1)
	e1:=b.GetEntry("key")
	e2:=b.Lookup()["key"]

	assert.Equal(t,*e1,*e2)
}