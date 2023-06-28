package coremiddleware

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type ContextKey string

const ContextKeyCorrelationID ContextKey = "x-correlation-id"

func Logger(l zerolog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := &responseObserver{ResponseWriter: w}
			rec := httptest.NewRecorder()

			ctx := r.Context()

			path := r.URL.EscapedPath()
			// reqData, _ := httputil.DumpRequest(r, false)

			cID := r.Header.Get(string(ContextKeyCorrelationID))
			if cID == "" {
				cID = uuid.New().String()
				r.Header.Set(string(ContextKeyCorrelationID), cID)
			}

			logger := l.With().Str(string(ContextKeyCorrelationID), cID).Logger()
			ctx = logger.WithContext(ctx)

			e := logger.Log().Str("path", path)
			defer func(start time.Time) {
				e.TimeDiff("latency", time.Now(), start).Send()
			}(time.Now())

			next.ServeHTTP(rec, r.WithContext(ctx))

			for k, v := range rec.Header() {
				ww.Header()[k] = v
			}

			if rec.Code == 0 {
				rec.Code = http.StatusOK
			}

			ww.WriteHeader(rec.Code)
			rec.Body.WriteTo(ww) //nolint
		})
	}
}

type responseObserver struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func (o *responseObserver) Write(p []byte) (n int, err error) {
	if !o.wroteHeader {
		o.WriteHeader(http.StatusOK)
	}
	n, err = o.ResponseWriter.Write(p)
	o.written += int64(n)
	return
}

func (o *responseObserver) WriteHeader(code int) {
	o.ResponseWriter.WriteHeader(code)
	if o.wroteHeader {
		return
	}
	o.wroteHeader = true
	o.status = code
}
