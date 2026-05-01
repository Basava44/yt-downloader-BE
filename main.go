package main

import (
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func isValidYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	format := r.URL.Query().Get("type")

	if url == "" || !isValidYouTubeURL(url) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	var args []string
	var filename string

	if format == "mp3" {
		args = []string{"-x", "--audio-format", "mp3", "-o", "-", url}
		filename = "audio.mp3"
	} else {
		args = []string{"-f", "mp4", "-o", "-", url}
		filename = "video.mp4"
	}

	cmd := exec.Command("yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Error creating pipe", 500)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, "Error starting process", 500)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, stdout)
	if err != nil {
		log.Println("Stream error:", err)
	}

	cmd.Wait()
}

func rateLimit(next http.Handler) http.Handler {
	limiter := make(map[string]time.Time)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if last, exists := limiter[ip]; exists {
			if time.Since(last) < 5*time.Second {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
		}

		limiter[ip] = time.Now()
		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/download", rateLimit(http.HandlerFunc(downloadHandler)))

	log.Println("Server running on :3000")
	http.ListenAndServe(":3000", mux)
}
