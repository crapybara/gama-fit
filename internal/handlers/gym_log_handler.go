package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gama-fit/database"
)

func HandleGymLogs(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}

		var content string
		err := database.DB.QueryRow("SELECT content FROM gym_logs WHERE user_id = $1 AND log_date = $2", userID, date).Scan(&content)
		if err == sql.ErrNoRows {
			content = ""
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(content))

	case http.MethodPost:
		date := r.FormValue("date")
		content := r.FormValue("content")

		if date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}

		_, err := database.DB.Exec(`
			INSERT INTO gym_logs (user_id, log_date, content)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, log_date)
			DO UPDATE SET content = EXCLUDED.content
		`, userID, date, content)

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Saved successfully"))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleGymLogDates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query("SELECT log_date FROM gym_logs WHERE user_id = $1 AND content != '' ORDER BY log_date ASC", userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err == nil {
			dates = append(dates, date)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dates)
}

func HandleExportGymLogs(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query("SELECT log_date, content FROM gym_logs WHERE user_id = $1 ORDER BY log_date DESC", userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("# GYM Logs Export\n\n")

	for rows.Next() {
		var date, content string
		if err := rows.Scan(&date, &content); err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", date, content))
	}

	w.Header().Set("Content-Disposition", "attachment; filename=gym_logs.md")
	w.Header().Set("Content-Type", "text/markdown")
	w.Write([]byte(sb.String()))
}
