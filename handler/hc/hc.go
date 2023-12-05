package hc

import (
	"encoding/json"
	"net/http"
	"time"
)

func Handler(version string) http.Handler {
	t := time.Now()
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "encoding/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": version,
			"uptime":  time.Since(t).String(),
		})
	}

	return http.HandlerFunc(fn)
}
