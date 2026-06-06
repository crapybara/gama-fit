package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gama-fit/database"
)

type FocusTask struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type FocusLog struct {
	Mode         string `json:"mode"`
	DurationMins int    `json:"duration_mins"`
	LocalDate    string `json:"local_date"`
	LocalTime    string `json:"local_time"`
}

func HandleFocusTasks(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)

	switch r.Method {
	case http.MethodGet:
		rows, err := database.DB.Query("SELECT id, title, completed FROM focus_tasks WHERE user_id = $1 ORDER BY id ASC", userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var tasks []FocusTask
		for rows.Next() {
			var t FocusTask
			var comp int
			if err := rows.Scan(&t.ID, &t.Title, &comp); err == nil {
				t.Completed = comp == 1
				tasks = append(tasks, t)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)

	case http.MethodPost:
		var t FocusTask
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if t.Title == "" {
			http.Error(w, "Title required", http.StatusBadRequest)
			return
		}
		_, err := database.DB.Exec("INSERT INTO focus_tasks (user_id, title) VALUES ($1, $2)", userID, t.Title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func HandleFocusLog(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var l FocusLog
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := database.DB.Exec("INSERT INTO focus_logs (user_id, mode, duration_mins, log_date, log_time) VALUES ($1, $2, $3, $4, $5)", userID, l.Mode, l.DurationMins, l.LocalDate, l.LocalTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func HandleFocusTaskActions(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	path := strings.TrimPrefix(r.URL.Path, "/api/focus/")
	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var t FocusTask
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		comp := 0
		if t.Completed {
			comp = 1
		}
		_, err := database.DB.Exec("UPDATE focus_tasks SET completed = $1 WHERE id = $2 AND user_id = $3", comp, id, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		_, err := database.DB.Exec("DELETE FROM focus_tasks WHERE id = $1 AND user_id = $2", id, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
