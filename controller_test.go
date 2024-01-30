package tupa

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkDirectAccessSendString(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := &TupaContext{
		request:  req,
		response: w,
	}

	start := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SendString(http.StatusOK, "Hello, World!")
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}
