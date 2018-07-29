package EventBus

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Log("New EventBus not created!")
		t.Fail()
	}
}

func TestHasCallback(t *testing.T) {
	bus := New()
	bus.Subscribe("topic", func() {})
	if bus.HasCallback("topic_topic") {
		t.Fail()
	}
	if !bus.HasCallback("topic") {
		t.Fail()
	}
}

func TestSubscribe(t *testing.T) {
	bus := New()
	if bus.Subscribe("topic", func() {}) != nil {
		t.Fail()
	}
	if bus.Subscribe("topic", "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnce(t *testing.T) {
	bus := New()
	if bus.SubscribeOnce("topic", func() {}) != nil {
		t.Fail()
	}
	if bus.SubscribeOnce("topic", "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnceAndManySubscribe(t *testing.T) {
	bus := New()
	event := "topic"
	flag := 0
	fn := func() { flag += 1 }
	bus.SubscribeOnce(event, fn)
	bus.Subscribe(event, fn)
	bus.Subscribe(event, fn)
	bus.Publish(event)

	if flag != 3 {
		t.Fail()
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	handler := func() {}
	bus.Subscribe("topic", handler)
	if bus.Unsubscribe("topic", handler) != nil {
		t.Fail()
	}
	if bus.Unsubscribe("topic", handler) == nil {
		t.Fail()
	}
}

func TestPublish(t *testing.T) {
	bus := New()
	bus.Subscribe("topic", func(a int, b int) {
		if a != b {
			t.Fail()
		}
	})
	bus.Publish("topic", 10, 10)
}

func TestSubcribeOnceAsync(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	bus.SubscribeOnceAsync("topic", func(a int, out *[]int) {
		*out = append(*out, a)
	})

	bus.Publish("topic", 10, &results)
	bus.Publish("topic", 10, &results)

	bus.WaitAsync()

	if len(results) != 1 {
		t.Fail()
	}

	if bus.HasCallback("topic") {
		t.Fail()
	}
}

func TestSubscribeAsyncTransactional(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	bus.SubscribeAsync("topic", func(a int, out *[]int, dur string) {
		sleep, _ := time.ParseDuration(dur)
		time.Sleep(sleep)
		*out = append(*out, a)
	}, true)

	bus.Publish("topic", 1, &results, "1s")
	bus.Publish("topic", 2, &results, "0s")

	bus.WaitAsync()

	if len(results) != 2 {
		t.Fail()
	}

	if results[0] != 1 || results[1] != 2 {
		t.Fail()
	}
}

func TestSubscribeAsync(t *testing.T) {
	results := make(chan int)

	bus := New()
	bus.SubscribeAsync("topic", func(a int, out chan<- int) {
		out <- a
	}, false)

	bus.Publish("topic", 1, results)
	bus.Publish("topic", 2, results)

	numResults := 0

	go func() {
		for _ = range results {
			numResults++
		}
	}()

	bus.WaitAsync()

	time.Sleep(10 * time.Millisecond)

	if numResults != 2 {
		t.Fail()
	}
}

func TestPublishHandler(t *testing.T) {
	results := make(chan int)

	publishHandler := &testPublishHandler{}

	bus := NewWithPublishHandler(publishHandler)
	bus.SubscribeAsync("topic", func(a int, out chan<- int) {
		out <- a
	}, false)

	bus.Publish("topic", 1, results)
	bus.Publish("topic", 2, results)

	numResults := 0

	go func() {
		for _ = range results {
			numResults++
		}
	}()

	bus.WaitAsync()

	time.Sleep(10 * time.Millisecond)

	if numResults != 2 {
		t.Fail()
	}

	assert.Equal(t, 2, publishHandler.NumPreEvents)
	assert.Equal(t, 2, publishHandler.NumEvents)
	assert.Equal(t, 0, publishHandler.Panics)
}

func TestPublishHandlerRecover(t *testing.T) {
	results := make(chan int)

	publishHandler := &testPublishHandler{}

	bus := NewWithPublishHandler(publishHandler)
	bus.SubscribeAsync("topic", func(a int, out chan<- int) {
		out <- a
	}, false)

	bus.Publish("topic", 1, results)
	bus.Publish("topic", 2, results)
	bus.Publish("topic", 3, nil)
	bus.Publish("topic", 4, 5, 6, results)
	bus.Publish("topic", 5)

	numResults := 0

	go func() {
		for _ = range results {
			numResults++
		}
	}()

	bus.WaitAsync()

	time.Sleep(10 * time.Millisecond)

	if numResults != 2 {
		t.Fail()
	}

	assert.Equal(t, 5, publishHandler.NumPreEvents)
	assert.Equal(t, 5, publishHandler.NumEvents)
	assert.Equal(t, 3, publishHandler.Panics)
}

type testPublishHandler struct {
	NumPreEvents int
	NumEvents    int
	Panics       int
}

func (tph *testPublishHandler) PrePublish(topic string, args ...interface{}) {
	tph.NumPreEvents++
}

func (tph *testPublishHandler) Publish(topic string, args []interface{}, callback reflect.Value, callArgs []reflect.Value) {
	defer func() {
		if r := recover(); r != nil {
			tph.Panics++
			fmt.Printf("Recovered from panic for EventBus publish: event=%s, with args=%v, error: %s\n", topic, args, r)
		}
	}()

	tph.NumEvents++
	callback.Call(callArgs)
}
