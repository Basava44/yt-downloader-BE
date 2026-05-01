package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func isValidYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	format := r.URL.Query().Get("type")

	if url == "" || !isValidYouTubeURL(url) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "ytdl-")
	if err != nil {
		http.Error(w, "Server error", 500)
		return
	}
	defer os.RemoveAll(tmpDir)

	var args []string
	var outName string

	if format == "mp3" {
		outName = "audio.mp3"
		args = []string{
			"-x", "--audio-format", "mp3",
			"-o", filepath.Join(tmpDir, outName),
			url,
		}
	} else {
		outName = "video.mp4"
		args = []string{
			"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
			"--merge-output-format", "mp4",
			"-o", filepath.Join(tmpDir, outName),
			url,
		}
	}

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Println("yt-dlp error:", err)
		http.Error(w, "Download failed", 500)
		return
	}

	outPath := filepath.Join(tmpDir, outName)

	w.Header().Set("Content-Disposition", "attachment; filename="+outName)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, outPath)
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
	mux.Handle("/download", corsMiddleware(rateLimit(http.HandlerFunc(downloadHandler))))

	log.Println("Server running on :3000")
	http.ListenAndServe(":3000", mux)
}
