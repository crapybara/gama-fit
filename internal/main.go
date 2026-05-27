package main

import (
	"log"
	"net/http"
	"os"

	"gama-fit/database"
	"gama-fit/handlers"
)

func main() {
	database.ConnectAndSetup()

	if os.Getenv("SEED_DEMO") == "1" {
		database.SeedDemoData()
	}

	// Core HTMX Endpoints
	http.HandleFunc("/api/goals", handlers.HandleGoals)
	http.HandleFunc("/api/goals/", handlers.HandleGoalActions)
	http.HandleFunc("/api/coins", handlers.GetCoins)
	http.HandleFunc("/api/checkins", handlers.HandleCheckins)
	http.HandleFunc("/api/score", handlers.GetFitnessScore)
	http.HandleFunc("/api/plans", handlers.HandleWorkoutPlan)
	http.HandleFunc("/api/freestyle", handlers.HandleFreestyle)
	http.HandleFunc("/api/creatine", handlers.HandleCreatine)
	http.HandleFunc("/api/streak", handlers.GetStreak)
//shop endpoints 
http.HandleFunc("/api/shop", handlers.HandleShop)
http.HandleFunc("/api/shop/", handlers.HandleShop)

	// Nutrition & Macro Endpoints
	http.HandleFunc("/api/macros/summary", handlers.HandleMacrosSummary)
	http.HandleFunc("/api/macros/targets", handlers.HandleSetTargets)
	http.HandleFunc("/api/meals", handlers.HandleMeals)

	// Sleep Endpoints
	http.HandleFunc("/api/sleep/summary", handlers.HandleSleepSummary)
	http.HandleFunc("/api/sleep", handlers.HandleSleep)
	http.HandleFunc("/api/sleep/history", handlers.HandleSleepHistory)

	// Analytics Endpoints
	http.HandleFunc("/api/analytics/metrics", handlers.HandleAnalytics)

	// DB Backup Endpoints
	http.HandleFunc("/api/db/export", handlers.HandleExportDB)
	http.HandleFunc("/api/db/import", handlers.HandleImportDB)

	// File Server for HTML/CSS/JS
	fs := http.FileServer(http.Dir("../external"))
	http.Handle("/", fs)

	log.Println("🚀 Gama Fitness Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
