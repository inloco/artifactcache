package main

import (
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type LoggedResponseWriter struct {
	ResponseWriter http.ResponseWriter
	StatusCode     int
}

func (w *LoggedResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *LoggedResponseWriter) Write(body []byte) (int, error) {
	return w.ResponseWriter.Write(body)
}
func (w *LoggedResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func logged(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		lrw := &LoggedResponseWriter{
			ResponseWriter: w,
		}

		t0 := time.Now()
		h(lrw, r, ps)
		t := time.Since(t0)

		log.Printf("%s %s - %d %v", r.Method, r.URL, lrw.StatusCode, t)
	}
}
