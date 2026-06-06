package main

import (
	"context"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	http.HandleFunc("/api/freestyle/marklog", handlers.AuthMiddleware(handlers.HandleMarkLog))
	http.HandleFunc("/api/cardio", handlers.AuthMiddleware(handlers.HandleCardio))
	http.HandleFunc("/api/bodyweight", handlers.AuthMiddleware(handlers.HandleBodyWeight))
	http.HandleFunc("/api/streak", handlers.AuthMiddleware(handlers.GetStreak))

	// shop endpoints
	http.HandleFunc("/api/shop", handlers.AuthMiddleware(handlers.HandleShop))
	http.HandleFunc("/api/shop/", handlers.AuthMiddleware(handlers.HandleShop))

	// Nutrition & Macro Endpoints (Protected)
	http.HandleFunc("/api/macros/summary", handlers.AuthMiddleware(handlers.HandleMacrosSummary))
	http.HandleFunc("/api/macros/targets", handlers.AuthMiddleware(handlers.HandleSetTargets))
	http.HandleFunc("/api/meals", handlers.AuthMiddleware(handlers.HandleMeals))
	http.HandleFunc("/api/foods/catalog", handlers.AuthMiddleware(handlers.HandleFoodCatalogJSON))

	// Focus Tasks (Study Mode)
	http.HandleFunc("/api/focus", handlers.AuthMiddleware(handlers.HandleFocusTasks))
	http.HandleFunc("/api/focus/log", handlers.AuthMiddleware(handlers.HandleFocusLog))
	http.HandleFunc("/api/focus/", handlers.AuthMiddleware(handlers.HandleFocusTaskActions))

	// Sleep Endpoints (Protected)
	http.HandleFunc("/api/sleep/summary", handlers.AuthMiddleware(handlers.HandleSleepSummary))
	http.HandleFunc("/api/sleep", handlers.AuthMiddleware(handlers.HandleSleep))
	http.HandleFunc("/api/sleep/history", handlers.AuthMiddleware(handlers.HandleSleepHistory))

	// Analytics Endpoints (Protected)
	http.HandleFunc("/analytics.html", handlers.AuthMiddleware(analytics.HandleAnalytics))
	http.HandleFunc("/api/analytics/muscle-1rm", handlers.AuthMiddleware(analytics.HandleMuscle1RM))
	http.HandleFunc("/api/analytics/heatmap", handlers.AuthMiddleware(analytics.HandleAnalyticsHeatmap))

	// Resource Handlers
	http.HandleFunc("/api/resources/gifs", handlers.AuthMiddleware(handlers.HandleGifs))
	http.HandleFunc("/api/resources/music", handlers.AuthMiddleware(handlers.HandleMusicList))
	http.HandleFunc("/api/resources/music-presets", handlers.AuthMiddleware(handlers.HandleMusicPresets))
	http.HandleFunc("/api/resources/videos", handlers.AuthMiddleware(handlers.HandleVideos))

	// DB Backup Endpoints (Protected)
	http.HandleFunc("/api/db/export", handlers.AuthMiddleware(handlers.HandleExportDB))
	http.HandleFunc("/api/db/import", handlers.AuthMiddleware(handlers.HandleImportDB))
	http.HandleFunc("/api/db/delete-all", handlers.AuthMiddleware(handlers.HandleDeleteAllData))

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

	server := &http.Server{
		Addr:         ":8095",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		localIP := "0.0.0.0"
		if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
			localIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
			conn.Close()
		}
		log.Printf("🚀 Gama Fitness Server running on http://%s:8095", localIP)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server stopped.")
}
