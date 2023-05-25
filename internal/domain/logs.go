package domain

import (
	"go.uber.org/zap"
	"net/http"
	"time"
)

type (
	responseData struct {
		status int
		size   int
	}

	requestData struct {
		uri       string
		method    string
		timeSpent time.Duration
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
		requestData  *requestData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WithLogging(h http.Handler, sugar *zap.SugaredLogger) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		logRespWriter := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   &responseData{},
			requestData:    &requestData{},
		}

		start := time.Now()

		logRespWriter.requestData.uri = r.RequestURI

		logRespWriter.requestData.method = r.Method

		h.ServeHTTP(&logRespWriter, r)

		logRespWriter.requestData.timeSpent = time.Since(start)

		sugar.Infoln(
			"uri", logRespWriter.requestData.uri,
			"method", logRespWriter.requestData.method,
			"duration", logRespWriter.requestData.timeSpent,
		)

		sugar.Infoln(
			"status", logRespWriter.responseData.status,
			"size", logRespWriter.responseData.size,
		)

	}

	return logFn
}
