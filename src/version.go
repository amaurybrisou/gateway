package src

import (
	"encoding/json"
	"net/http"
	"time"
)

var (
	BuildVersion, BuildHash, BuildTime string = "1.0", "localhost", time.Now().String()
)

func Version(w http.ResponseWriter, r *http.Request) {
	// Create a BuildInfo struct with the desired values
	buildInfo := struct {
		BuildVersion string `json:"build_version"`
		BuildHash    string `json:"build_hash"`
		BuildTime    string `json:"build_time"`
	}{
		BuildVersion: BuildVersion,
		BuildHash:    BuildHash,
		BuildTime:    BuildTime,
	}

	// Set the content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Encode the BuildInfo struct as JSON
	json.NewEncoder(w).Encode(buildInfo) //nolint
}
