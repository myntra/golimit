package bucket

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestEntry(t *testing.T) {

	//Expired entry check
	expiry:=time.Now().UnixNano()
	count :=int32(1)

	e:=NewEntry(count,expiry)
	assert.Equal(t,expiry,e.expires)
	assert.Equal(t, count,e.Count())
	assert.True(t, e.Expiry()<time.Now().UnixNano())
	assert.Equal(t,true,e.Expired())
	assert.True(t, e.LastModified()<time.Now().UnixNano())

	e.ReInit(1,time.Now().UnixNano()+10000)

	e.Incr(2)

	count +=2

	assert.Equal(t, count,e.Count())

	e.Sync(5)
	count +=5
	assert.Equal(t, count,e.Count())


}



