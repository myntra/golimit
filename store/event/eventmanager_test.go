package event

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"strconv"
)

type dummyHandler struct {
	ch chan Event
}

func (dh *dummyHandler) Handle(e Event) {
	dh.ch <- e
}

func TestGoLimitEventManager(t *testing.T) {
	mgr := GetMgrInstance()

	ch := make(chan Event, 10)
	dh := &dummyHandler{ch: ch}

	mgr.RegisterHandler(KEYEVENT, dh)

	e := mgr.GetPool(KEYEVENT).Get().(*KeyEvent)
	e.Key = "test1"
	e.Count = 5
	e.Allowed = true
	e.Time = time.Now().UnixNano()
	mgr.Publish(e)
	e2 := <-ch
	assert.Equal(t, e2.(*KeyEvent).Key, e.Key)

	assert.Equal(t, e.GetRoute(), KEYEVENT)

	mgr.UnRegisterHandler(KEYEVENT, dh)

	mgr.Publish(e2)

	assert.True(t, nil == mgr.GetPool("unknown"))
	mgr.GetPool(KEYEVENT).Put(e2)
}

func TestGetMgrInstance(t *testing.T) {
	mgr:= GetMgrInstanceWithParam(0,10)

	ch := make(chan Event, 10)
	dh := &dummyHandler{ch: ch}
	mgr.RegisterHandler(KEYEVENT, dh)

	for i:=0;i<10;i++{
		e := mgr.GetPool(KEYEVENT).Get().(*KeyEvent)
		e.Key = "test"+strconv.Itoa(i)
		e.Count = 5
		e.Allowed = true
		e.Time = time.Now().UnixNano()
		assert.True(t,mgr.Publish(e))
	}
	e := mgr.GetPool(KEYEVENT).Get().(*KeyEvent)
	e.Key = "test10"
	e.Count = 5
	e.Allowed = true
	e.Time = time.Now().UnixNano()
	assert.False(t,mgr.Publish(e))

}