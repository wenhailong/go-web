package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Decorator func(http.HandlerFunc) http.HandlerFunc

func loggingAndRespError() Decorator {
	return func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err, ok := recover().(error); ok {
					log.Error(err.Error())

					responseError(w, err)
				}
			}()

			fn(w, r)
		}
	}
}

func counting(c *Counter) Decorator {
	return func(fn http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func(start time.Time) {
				c.AddLatency(time.Since(start).Nanoseconds())

			}(time.Now())

			c.AddRequest(1)
			fn(w, r)
		}
	}
}

func Decorate(fn http.HandlerFunc, ds ...Decorator) http.HandlerFunc {
	decorated := fn
	for _, decorate := range ds {
		decorated = decorate(decorated)
	}

	return decorated
}
