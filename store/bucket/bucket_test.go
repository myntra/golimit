package bucket

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestBucket(t *testing.T) {
	b:=NewKeyBucket()
	blocked,expiry,justbreached:=b.Incr("one",1,10,1)

	for i:=0;i<9&& expiry>time.Now().UnixNano();i++{
		blocked,expiry,justbreached=b.Incr("one",1,10,1)
		assert.False(t,blocked)
		assert.False(t,justbreached)
	}

	blocked,expiry,justbreached=b.Incr("one",1,10,1)
	assert.True(t,blocked)
	assert.True(t,justbreached)

	time.Sleep(time.Second)

	blocked,expiry,justbreached=b.Incr("one",1,10,1)
	assert.False(t,blocked)
	assert.False(t,justbreached)


	b.Sync("one",5,expiry)
	e:=b.GetEntry("one")
	assert.Equal(t,int32(6),e.Count())


	b.Sync("two",5,expiry)
	e=b.GetEntry("two")
	assert.Equal(t,int32(5),e.Count())

	time.Sleep(time.Second)

	b.Sync("two",5,expiry)
	e=b.GetEntry("two")
	assert.Equal(t,int32(5),e.Count())


	b.Incr("three",1,10,0)
	e=b.GetEntry("three")
	assert.Equal(t,int32(1),e.Count())
	assert.True(t,e.Expired())

}

