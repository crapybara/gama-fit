package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"gama-fit/database"
)

// --- 1. FREESTYLE TRACKER ---
func HandleFreestyle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("action") == "options" {
		rows, err := database.DB.Query("SELECT DISTINCT exercise_name FROM workout_plans WHERE exercise_name != '' ORDER BY exercise_name ASC")
		if err != nil {
			w.Write([]byte("<option disabled>No exercises found</option>"))
			return
		}
		defer rows.Close()

		html := `<option value="" disabled selected>Choose from your plan...</option>`
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				html += fmt.Sprintf(`<option value="%s">%s</option>`, name, name)
			}
		}
		w.Write([]byte(html))
		return
	}

	if r.Method == "POST" {
		exercise := r.FormValue("exercise")
		weightStr := r.FormValue("weight")
		repsStr := r.FormValue("reps")
		if exercise != "" && weightStr != "" && repsStr != "" {
			weight, err1 := strconv.ParseFloat(weightStr, 64)
			reps, err2 := strconv.Atoi(repsStr)
			if err1 == nil && err2 == nil {
				_, _ = database.DB.Exec("INSERT INTO freestyle_logs (exercise_name, weight, reps, logged_date, logged_time) VALUES (?, ?, ?, date('now'), time('now'))", exercise, weight, reps)
			}
		}
	} else if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		_, _ = database.DB.Exec("DELETE FROM freestyle_logs WHERE id = ?", id)
	}

	rows, err := database.DB.Query("SELECT id, exercise_name, weight, reps FROM freestyle_logs WHERE logged_date = date('now') ORDER BY id ASC")
	if err != nil {
		w.Write([]byte(""))
		return
	}
	defer rows.Close()

	html := ""
	for rows.Next() {
		var id int
		var name string
		var weight float64
		var reps int
		if err := rows.Scan(&id, &name, &weight, &reps); err == nil {
			html += fmt.Sprintf(`
		<div class="flex items-center justify-between bg-zinc-900/40 border border-white/5 rounded-xl p-3 mb-2 hover:bg-zinc-900/80 transition-colors">
			<span class="text-white text-sm font-bold">%s</span>
			<div class="flex items-center gap-3 text-xs font-mono">
				<span class="text-zinc-400">%v KG</span>
				<span class="text-zinc-600">×</span>
				<span class="text-zinc-400">%d Reps</span>
				<button hx-delete="/api/freestyle?id=%d" hx-target="#freestyle-list" class="ml-2 text-zinc-600 hover:text-red-500"><svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
			</div>
		</div>`, name, weight, reps, id)
		}
	}

	w.Write([]byte(html))
}

// --- 2. WORKOUT PLAN (7-Day System) ---
func HandleWorkoutPlan(w http.ResponseWriter, r *http.Request) {
	dayStr := r.URL.Query().Get("day")
	if dayStr == "" {
		dayStr = "1"
	}
	day, _ := strconv.Atoi(dayStr)

	if r.Method == "POST" {
		var count int
		_ = database.DB.QueryRow("SELECT COUNT(*) FROM workout_plans WHERE day_of_week = ?", day).Scan(&count)

		if count < 15 {
			exercise := r.FormValue("exercise")
			setsStr := r.FormValue("sets")
			reps := r.FormValue("reps")

			sets, err := strconv.Atoi(setsStr)
			if err != nil || sets <= 0 {
				sets = 3
			}

			if exercise != "" {
				_, _ = database.DB.Exec("INSERT INTO workout_plans (day_of_week, exercise_name, sets, reps) VALUES (?, ?, ?, ?)", day, exercise, sets, reps)
			}
		}
	} else if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		_, _ = database.DB.Exec("DELETE FROM workout_plans WHERE id = ?", id)
	}

	rows, err := database.DB.Query("SELECT id, exercise_name, sets, reps FROM workout_plans WHERE day_of_week = ? ORDER BY id ASC", day)
	if err != nil {
		w.Write([]byte(`<div class="text-zinc-500 text-sm text-center py-6">Rest day or no exercises added yet.</div>`))
		return
	}
	defer rows.Close()

	exerciseCount := 0
	html := `<div class="space-y-2 mt-4">`
	for rows.Next() {
		exerciseCount++
		var id int
		var name, reps string
		var sets int
		if err := rows.Scan(&id, &name, &sets, &reps); err == nil {
			html += fmt.Sprintf(`
		<div class="flex items-center justify-between bg-zinc-900/40 border border-white/5 rounded-xl p-3 hover:bg-zinc-900/80 transition-colors">
			<span class="text-zinc-300 text-sm font-medium">%s</span>
			<div class="flex items-center gap-2 sm:gap-3">
				<span class="text-[10px] font-bold uppercase tracking-wider text-blue-400 bg-blue-400/10 px-2 py-1 rounded">%d Sets</span>
				<span class="text-[10px] font-bold uppercase tracking-wider text-zinc-500 bg-zinc-800 px-2 py-1 rounded">%s Reps</span>
				<button hx-delete="/api/plans?day=%d&id=%d" hx-target="#plans-container" class="text-zinc-600 hover:text-red-500 transition-colors"><svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
			</div>
		</div>`, name, sets, reps, day, id)
		}
	}
	html += `</div>`

	if exerciseCount == 0 {
		html = `<div class="text-zinc-500 text-sm text-center py-6">Rest day or no exercises added yet.</div>`
	}

	formHtml := ""
	if exerciseCount >= 15 {
		formHtml = `<div class="text-app-pink text-xs font-bold text-center mt-4">Max limit of 15 exercises reached for this day.</div>`
	} else {
		formHtml = fmt.Sprintf(`
		<form hx-post="/api/plans?day=%d" hx-target="#plans-container" hx-on::after-request="this.reset()" class="flex flex-col sm:flex-row gap-3 mt-6">
			<input type="text" name="exercise" placeholder="Exercise name..." required class="flex-[3] bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-pink transition-colors">
			<div class="flex gap-2 flex-1">
				<input type="number" name="sets" placeholder="Sets" value="3" min="1" max="10" required class="flex-1 min-w-0 bg-zinc-900/50 border border-zinc-700 rounded-xl px-2 py-3 text-sm text-center text-white outline-none focus:border-app-pink transition-colors font-mono">
				<input type="text" name="reps" placeholder="Reps" value="8-10" required class="flex-1 min-w-0 bg-zinc-900/50 border border-zinc-700 rounded-xl px-2 py-3 text-sm text-center text-white outline-none focus:border-app-pink transition-colors font-mono">
				<button type="submit" class="bg-app-pink text-white font-bold px-5 rounded-xl hover:bg-pink-500 transition-all shadow-[0_0_15px_rgba(255,0,160,0.2)] flex items-center justify-center shrink-0">
					<svg class="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
				</button>
			</div>
		</form>`, day)
	}

	w.Write([]byte(html + formHtml))
}

// --- 3. BODY WEIGHT TRACKER ---
func HandleBodyWeight(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
		if weight > 0 {
			_, _ = database.DB.Exec("INSERT INTO body_weight_logs (weight, log_date) VALUES (?, date('now')) ON CONFLICT(log_date) DO UPDATE SET weight=excluded.weight", weight)
		}
	} else if r.Method == http.MethodDelete {
		_, _ = database.DB.Exec("DELETE FROM body_weight_logs WHERE log_date = date('now')")
	}

	var todayWeight float64
	err := database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE log_date = date('now')").Scan(&todayWeight)

	if err == nil {
		fmt.Fprintf(w, `
			<div id="bodyweight-content" class="relative z-10 flex-1 flex flex-col items-center justify-center py-4 w-full">
				<div class="flex flex-col items-center justify-center transition-all duration-300 w-full">
					<span class="font-display text-5xl sm:text-6xl font-black text-white group-hover:text-blue-400 transition-colors">%.1f<span class="text-2xl sm:text-3xl text-blue-400 ml-1">kg</span></span>
					<span class="text-[10px] sm:text-xs uppercase font-bold text-zinc-500 tracking-wider mt-2">Today's Weight</span>
					<button hx-delete="/api/bodyweight" hx-swap="outerHTML" hx-target="#bodyweight-content" class="mt-4 bg-zinc-900 border border-zinc-700 hover:border-red-500 text-zinc-400 hover:text-red-500 px-4 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest transition-colors">Reset</button>
				</div>
			</div>
		`, todayWeight)
	} else {
		fmt.Fprint(w, `
			<div id="bodyweight-content" class="relative z-10 flex-1 flex flex-col items-center justify-center py-4 w-full">
				<form hx-post="/api/bodyweight" hx-swap="outerHTML" hx-target="#bodyweight-content" class="flex flex-col items-center justify-center gap-3 w-full max-w-[200px] mx-auto px-4">
					<input type="number" name="weight" step="0.1" placeholder="00.0" required class="w-full bg-zinc-900/80 border border-zinc-700 rounded-xl px-4 py-3 text-2xl sm:text-3xl text-center text-white outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 transition-all font-mono">
					<button type="submit" class="w-full bg-blue-500 text-black font-black py-3 rounded-xl hover:bg-blue-400 transition-all shadow-[0_0_15px_rgba(59,130,246,0.3)] hover:shadow-[0_0_25px_rgba(59,130,246,0.5)] uppercase tracking-wider text-xs">Log Weight</button>
				</form>
			</div>
		`)
	}
}
