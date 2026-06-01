package main

import (
	"log"
	"mime"
	"net/http"
	"strings"

	"gama-fit/analytics"
	"gama-fit/database"
	"gama-fit/handlers"
)

func init() {
	// Register common MIME types to ensure they are served correctly
	mime.AddExtensionType(".mp4", "video/mp4")
	mime.AddExtensionType(".webm", "video/webm")
	mime.AddExtensionType(".ogg", "video/ogg")
	mime.AddExtensionType(".mp3", "audio/mpeg")
	mime.AddExtensionType(".wav", "audio/wav")
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")
}

func main() {
	database.ConnectAndSetup()
	handlers.InitTemplates()

	// Auth Endpoints
	http.HandleFunc("/api/register", handlers.HandleRegister)
	http.HandleFunc("/api/login", handlers.HandleLogin)
	http.HandleFunc("/api/user", handlers.AuthMiddleware(handlers.HandleCurrentUser))

	// User Settings
	http.HandleFunc("/api/settings", handlers.AuthMiddleware(handlers.HandleSettings))
	http.HandleFunc("/api/settings/theme", handlers.AuthMiddleware(handlers.HandleTheme))
	http.HandleFunc("/api/settings/pomo", handlers.AuthMiddleware(handlers.GetPomoSettings))

	// Gym Logs
	http.HandleFunc("/api/logs", handlers.AuthMiddleware(handlers.HandleGymLogs))
	http.HandleFunc("/api/logs/export", handlers.AuthMiddleware(handlers.HandleExportGymLogs))

	// Core HTMX Endpoints (Protected)
	http.HandleFunc("/api/goals", handlers.AuthMiddleware(handlers.HandleGoals))
	http.HandleFunc("/api/goals/", handlers.AuthMiddleware(handlers.HandleGoalActions))
	http.HandleFunc("/api/coins", handlers.AuthMiddleware(handlers.GetCoins))
	http.HandleFunc("/api/checkins", handlers.AuthMiddleware(handlers.HandleCheckins))
	http.HandleFunc("/api/score", handlers.AuthMiddleware(handlers.GetFitnessScore))
	http.HandleFunc("/api/plans", handlers.AuthMiddleware(handlers.HandleWorkoutPlan))
	http.HandleFunc("/api/plans/heatmap", handlers.AuthMiddleware(handlers.HandleWorkoutHeatmap))
	http.HandleFunc("/api/freestyle", handlers.AuthMiddleware(handlers.HandleFreestyle))
	http.HandleFunc("/api/bodyweight", handlers.AuthMiddleware(handlers.HandleBodyWeight))
	http.HandleFunc("/api/streak", handlers.AuthMiddleware(handlers.GetStreak))

	// shop endpoints
	http.HandleFunc("/api/shop", handlers.AuthMiddleware(handlers.HandleShop))
	http.HandleFunc("/api/shop/", handlers.AuthMiddleware(handlers.HandleShop))

	// Nutrition & Macro Endpoints (Protected)
	http.HandleFunc("/api/macros/summary", handlers.AuthMiddleware(handlers.HandleMacrosSummary))
	http.HandleFunc("/api/macros/targets", handlers.AuthMiddleware(handlers.HandleSetTargets))
	http.HandleFunc("/api/meals", handlers.AuthMiddleware(handlers.HandleMeals))

	// Sleep Endpoints (Protected)
	http.HandleFunc("/api/sleep/summary", handlers.AuthMiddleware(handlers.HandleSleepSummary))
	http.HandleFunc("/api/sleep", handlers.AuthMiddleware(handlers.HandleSleep))
	http.HandleFunc("/api/sleep/history", handlers.AuthMiddleware(handlers.HandleSleepHistory))

	// Analytics Endpoints (Protected)
	http.HandleFunc("/analytics.html", handlers.AuthMiddleware(analytics.HandleAnalytics))

	// Resource Handlers
	http.HandleFunc("/api/resources/gifs", handlers.AuthMiddleware(handlers.HandleGifs))
	http.HandleFunc("/api/resources/music", handlers.AuthMiddleware(handlers.HandleMusicList))
	http.HandleFunc("/api/resources/music-presets", handlers.AuthMiddleware(handlers.HandleMusicPresets))
	http.HandleFunc("/api/resources/videos", handlers.AuthMiddleware(handlers.HandleVideos))

	// DB Backup Endpoints (Protected)
	http.HandleFunc("/api/db/export", handlers.AuthMiddleware(handlers.HandleExportDB))
	http.HandleFunc("/api/db/import", handlers.AuthMiddleware(handlers.HandleImportDB))

	// File Server for HTML/CSS/JS
	fs := http.FileServer(http.Dir("../external"))

	// Custom handler to protect static files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Allow public access to login, register, and static assets
		if r.URL.Path == "/login.html" || r.URL.Path == "/register.html" ||
			strings.HasPrefix(r.URL.Path, "/styling/") ||
			strings.HasPrefix(r.URL.Path, "/scripts/") ||
			strings.HasPrefix(r.URL.Path, "/assets/") ||
			r.URL.Path == "/favicon.ico" {
			fs.ServeHTTP(w, r)
			return
		}

		// Protect other files (index.html, workout.html, etc.)
		_, err := handlers.GetUserID(r)
		if err != nil {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return
		}
		fs.ServeHTTP(w, r)
	})

	log.Println("🚀 Gama Fitness Server running on http://0.0.0.0:8080")
	http.ListenAndServe(":8080", nil)
}
