package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const gifDir = "../external/assets/gifs"
const videoDir = "../external/assets/videos"
const musicListDir = "../external/assets/music_list"

func HandleGifs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		files, err := os.ReadDir(gifDir)
		if err != nil {
			os.MkdirAll(gifDir, 0755)
		}

		var gifs []string
		for _, f := range files {
			if !f.IsDir() {
				gifs = append(gifs, f.Name())
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gifs)

	case http.MethodPost:
		file, header, err := r.FormFile("gif")
		if err != nil {
			http.Error(w, "Upload failed", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := header.Filename
		dst, err := os.Create(filepath.Join(gifDir, filename))
		if err != nil {
			http.Error(w, "Failed to save gif", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		filename := r.URL.Query().Get("name")
		if filename == "" {
			http.Error(w, "Name required", http.StatusBadRequest)
			return
		}
		err := os.Remove(filepath.Join(gifDir, filename))
		if err != nil {
			http.Error(w, "Delete failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleMusicList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		preset := r.URL.Query().Get("preset")
		if preset == "" {
			preset = "music.txt"
		}

		tracks := loadMusicPreset(preset)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tracks)

	case http.MethodPost:
		file, header, err := r.FormFile("preset")
		if err != nil {
			http.Error(w, "Preset upload failed", http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := filepath.Base(header.Filename)
		if !strings.EqualFold(filepath.Ext(filename), ".txt") {
			http.Error(w, "Only .txt presets are allowed", http.StatusBadRequest)
			return
		}

		if err := os.MkdirAll(musicListDir, 0755); err != nil {
			http.Error(w, "Failed to prepare music list directory", http.StatusInternalServerError)
			return
		}

		dst, err := os.Create(filepath.Join(musicListDir, filename))
		if err != nil {
			http.Error(w, "Failed to save preset", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to write preset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"preset": filename})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleMusicPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(musicListDir)
	if err != nil {
		_ = os.MkdirAll(musicListDir, 0755)
		files = nil
	}

	presets := []string{}
	for _, f := range files {
		if !f.IsDir() && strings.EqualFold(filepath.Ext(f.Name()), ".txt") {
			presets = append(presets, f.Name())
		}
	}
	sort.Strings(presets)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presets)
}

func HandleVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(videoDir)
	if err != nil {
		_ = os.MkdirAll(videoDir, 0755)
		files = nil
	}

	videos := []string{}
	for _, f := range files {
		if !f.IsDir() {
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext == ".mp4" || ext == ".webm" {
				videos = append(videos, f.Name())
			}
		}
	}
	sort.Strings(videos)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

func loadMusicPreset(preset string) []string {
	defaultLinks := []string{
		"https://stream.chillhop.com/mp3/9476",
		"https://ia601004.us.archive.org/31/items/lofigirl/2-AM-Study-Session/01%20hoogway%20-%20Missing%20Earth%20%28Kupla%20Master%29.mp3",
	}

	filename := filepath.Base(preset)
	if !strings.EqualFold(filepath.Ext(filename), ".txt") {
		filename = "music.txt"
	}

	paths := []string{
		filepath.Join(musicListDir, filename),
		"../music.txt",
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if tracks := parseMusicPreset(string(content)); len(tracks) > 0 {
			return tracks
		}
	}

	return defaultLinks
}

func parseMusicPreset(content string) []string {
	lines := strings.Split(content, "\n")
	var tracks []string
	baseURL := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			if strings.HasSuffix(line, "/") {
				baseURL = line
				continue
			}
			tracks = append(tracks, line)
			continue
		}

		if baseURL != "" {
			tracks = append(tracks, baseURL+strings.TrimPrefix(line, "/"))
			continue
		}

		tracks = append(tracks, line)
	}

	return tracks
}
