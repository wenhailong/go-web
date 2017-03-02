package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
)

const CounterLimit = 1 << 62

type Counter struct {
	request          int64
	latency          int64
	requestPerSecond int64
}

var g_counter Counter

func (p *Counter) AddRequest(n int64) {
	if p.Request() > CounterLimit {
		p.Clean()
	}

	atomic.AddInt64(&p.request, n)
}

func (p *Counter) Request() int64 {
	return atomic.LoadInt64(&p.request)
}

func (p *Counter) AddLatency(n int64) {
	if p.Latency() > CounterLimit {
		p.Clean()
	}

	atomic.AddInt64(&p.latency, n)
}

func (p *Counter) Latency() int64 {
	return atomic.LoadInt64(&p.latency)
}

func (p *Counter) AveLatency() int64 {
	requests := p.Request()
	if requests == 0 {
		return 0
	} else {
		return p.Latency() / requests
	}
}

func (p *Counter) SetRequestPerSecond(n int64) {
	atomic.SwapInt64(&p.requestPerSecond, n)
}

func (p *Counter) RequestPerSecond() int64 {
	return atomic.LoadInt64(&p.requestPerSecond)
}

func (p *Counter) Clean() {
	atomic.SwapInt64(&p.request, 0)
	atomic.SwapInt64(&p.latency, 0)
	atomic.SwapInt64(&p.requestPerSecond, 0)
}

func counterHander(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"request":          g_counter.Request(),
		"latency":          g_counter.Latency(),
		"aveLatency":       g_counter.AveLatency(),
		"requestPerSecond": g_counter.RequestPerSecond(),
	}

	bResult, err := json.Marshal(result)
	if err != nil {
		err := NewError("[counterHander] json.Marshal failed. error=%v", err)
		checkError(err)
	}

	io.WriteString(w, string(bResult))
}
