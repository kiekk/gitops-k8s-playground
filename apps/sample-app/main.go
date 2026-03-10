package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"app":         "sample-app",
		"version":     getEnv("APP_VERSION", "0.0.1"),
		"environment": getEnv("APP_ENV", "development"),
		"hostname":    hostname(),
		"goVersion":   runtime.Version(),
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	keys := []string{"APP_ENV", "APP_VERSION", "DB_HOST", "DB_PASSWORD"}
	info := make(map[string]string, len(keys))
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			v = "(not set)"
		}
		if k == "DB_PASSWORD" && v != "(not set)" {
			v = "****"
		}
		info[k] = v
	}
	writeJSON(w, http.StatusOK, info)
}

func handleStressCPU(w http.ResponseWriter, r *http.Request) {
	duration := 30 * time.Second
	done := make(chan struct{})

	go func() {
		deadline := time.Now().Add(duration)
		for time.Now().Before(deadline) {
			// tight loop
		}
		close(done)
	}()

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "started",
		"duration": duration.String(),
	})

	<-done
}

func handleStressMemory(w http.ResponseWriter, r *http.Request) {
	const chunkSize = 10 * 1024 * 1024 // 10MB per chunk
	var sink [][]byte

	for i := 0; i < 100; i++ {
		chunk := make([]byte, chunkSize)
		for j := range chunk {
			chunk[j] = byte(j % 256)
		}
		sink = append(sink, chunk)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "allocated",
		"allocated": fmt.Sprintf("%d MB", len(sink)*chunkSize/(1024*1024)),
	})

	runtime.KeepAlive(sink)
}

func main() {
	port := getEnv("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)
	mux.HandleFunc("/info", handleInfo)
	mux.HandleFunc("/stress/cpu", handleStressCPU)
	mux.HandleFunc("/stress/memory", handleStressMemory)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("sample-app %s starting on :%s", getEnv("APP_VERSION", "0.0.1"), port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
