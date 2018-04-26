package event

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

type dummyHandler struct {
	ch chan Event
}

func(dh * dummyHandler) Handle(e Event){
	dh.ch<-e
}

func TestGoLimitEventManager(t *testing.T) {
	mgr:=GetMgrInstance()

	ch:= make(chan Event,10)
	dh:=&dummyHandler{ch:ch}

	mgr.RegisterHandler(KEYEVENT, dh)

	e:= mgr.GetPool(KEYEVENT).Get().(*KeyEvent)
	e.Key="test1"
	e.Count=5
	e.Allowed=true
	e.Time=time.Now().UnixNano()
	mgr.Publish(e)
	e2:=<-ch
	assert.Equal(t,e2.(*KeyEvent).Key,e.Key)

	assert.Equal(t,e.GetRoute(),KEYEVENT)

	mgr.UnRegisterHandler(KEYEVENT,dh)


	mgr.Publish(e2)

	assert.True(t,nil==mgr.GetPool("unknown"))
	mgr.GetPool(KEYEVENT).Put(e2)
}