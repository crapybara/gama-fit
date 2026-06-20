package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"gama-fit/database"
)

func HandleExportDB(w http.ResponseWriter, r *http.Request) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5433/gamafit?sslmode=disable"
	}

	var stdout, stderr bytes.Buffer
	// --clean --if-exists ensures the backup can be imported over an existing database
	cmd := exec.Command("pg_dump", "--clean", "--if-exists", dsn)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		log.Printf("Export error: %v\nStderr: %s", err, stderr.String())
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Export failed. Check server logs.\n\nDetails: %v\n%s", err, stderr.String())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gamafit_backup_%s.sql", time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "application/sql")
	w.Write(stdout.Bytes())
}

func HandleImportDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("database")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "gamafit_import_*.sql")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	if _, err := tempFile.ReadFrom(file); err != nil {
		http.Error(w, "Failed to save temp file", http.StatusInternalServerError)
		return
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5433/gamafit?sslmode=disable"
	}

	var stderr bytes.Buffer
	cmd := exec.Command("psql", dsn, "-f", tempPath)
	cmd.Stderr = &stderr
	err = cmd.Run()

	if err != nil {
		log.Printf("Import error: %v\nStderr: %s", err, stderr.String())
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Import failed. Ensure the .sql file is a valid backup.\n\nDetails: %v\n%s", err, stderr.String())
		return
	}

	// Success: Redirect back to settings
	http.Redirect(w, r, "/settings.html?import=success", http.StatusSeeOther)
}

func HandleDeleteAllData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := GetUserID(r)

	// Transactional delete of all user data
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	tables := []string{
		"workout_plans", "freestyle_logs", "body_weight_logs",
		"cardio_logs", "gym_logs", "user_macros_final",
		"daily_meals", "food_catalog", "sleep_logs",
		"checkins", "goals", "shop_catalog",
	}

	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE user_id = $1", table), userID)
		if err != nil {
			tx.Rollback()
			log.Printf("Error clearing %s: %v", table, err)
			http.Error(w, "Error clearing "+table, http.StatusInternalServerError)
			return
		}
	}

	// Reset stats but keep the user record
	_, err = tx.Exec("UPDATE user_stats SET bmi=0, height=0, neck=0, belly=0, arms=0, calf=0, age=25, goal_weight=0, total_coins=0, current_streak=0 WHERE user_id = $1", userID)
	if err != nil {
		tx.Rollback()
		http.Error(w, "Error resetting stats", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Location", "/settings.html?reset=true")
	w.WriteHeader(http.StatusOK)
}
