package bucket

import (
	"time"
	"sync/atomic"
	"sync"

)

type KeyEntry struct{
	sync.RWMutex
	expires             	int64
	lastmodified         	int64
	count			int32
}

func (e*KeyEntry) Count() int32{
	return atomic.LoadInt32(&e.count)
}


func (e*KeyEntry) LastModified() int64{
	return atomic.LoadInt64(&e.lastmodified)
}

func (e*KeyEntry) Expiry() int64{
	return atomic.LoadInt64(&e.expires)
}

func NewEntry(count int32, expires int64) *KeyEntry {
	e := &KeyEntry{count:count, expires:expires,
		lastmodified:time.Now().UnixNano()}
	return e
}

func (e*KeyEntry)ReInit(count int32, expires int64){
	atomic.StoreInt64(&e.lastmodified,time.Now().UnixNano())
	atomic.StoreInt64(&e.expires,expires)
	atomic.StoreInt32(&e.count,count)
}

func (e*KeyEntry) Expired() bool{
	expires := e.Expiry()
	return expires < time.Now().UnixNano()
}

/**
	return total count and curr second count
 */
func (e*KeyEntry) Incr(count int32) int32{
	atomic.StoreInt64(&e.lastmodified,time.Now().UnixNano())
	return atomic.AddInt32(&e.count,count)
}

func (e*KeyEntry) Sync(count int32) int32{
	return atomic.AddInt32(&e.count,count)
}

