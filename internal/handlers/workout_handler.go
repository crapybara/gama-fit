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
			<div class="flex items-center gap-3">
				<span class="text-[10px] font-bold uppercase tracking-wider text-app-blue bg-app-blue/10 px-2 py-1 rounded">%d Sets</span>
				<span class="text-[10px] font-bold uppercase tracking-wider text-zinc-500 bg-zinc-800 px-2 py-1 rounded">%s Reps</span>
				<button hx-delete="/api/plans?day=%d&id=%d" hx-target="#plans-container" class="text-zinc-600 hover:text-red-500"><svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
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
		<form hx-post="/api/plans?day=%d" hx-target="#plans-container" hx-on::after-request="this.reset()" class="flex gap-2 mt-5">
			<input type="text" name="exercise" placeholder="Exercise..." required class="flex-1 bg-zinc-900/50 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-white outline-none focus:border-app-pink">
			<input type="number" name="sets" placeholder="Sets" value="3" min="1" max="10" required class="w-16 bg-zinc-900/50 border border-zinc-700 rounded-lg px-2 py-2 text-sm text-center text-white outline-none focus:border-app-pink">
			<input type="text" name="reps" placeholder="e.g. 8-10" value="8-10" required class="w-20 bg-zinc-900/50 border border-zinc-700 rounded-lg px-2 py-2 text-sm text-center text-white outline-none focus:border-app-pink">
			<button type="submit" class="bg-app-pink text-white font-bold px-4 rounded-lg hover:bg-pink-500 transition-all pink-glow-btn">+</button>
		</form>`, day)
	}

	w.Write([]byte(html + formHtml))
}

// --- 3. CREATINE LOGGING ---
func HandleCreatine(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _ = database.DB.Exec("INSERT OR IGNORE INTO creatine_tracker_final (log_date) VALUES (date('now'))")
	}

	var taken bool
	_ = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM creatine_tracker_final WHERE log_date = date('now'))").Scan(&taken)

	if taken {
		fmt.Fprint(w, `
			<button disabled class="w-28 h-28 rounded-full border-4 border-blue-500 bg-blue-500/20 flex flex-col items-center justify-center transition-all duration-300 shadow-[0_0_40px_rgba(59,130,246,0.3)] cursor-not-allowed">
				<div class="flex flex-col items-center justify-center text-blue-400 animate-[bounce_0.5s_ease-in-out]">
					<svg class="w-10 h-10 drop-shadow-[0_0_8px_rgba(96,165,250,0.8)]" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
				</div>
			</button>
		`)
	} else {
		fmt.Fprint(w, `
			<button hx-post="/api/creatine" hx-swap="outerHTML" onclick="playPowerSound()" class="w-28 h-28 rounded-full border-4 border-zinc-800 bg-zinc-900/50 flex flex-col items-center justify-center hover:border-blue-500 hover:bg-blue-500/10 transition-all duration-300 group shadow-[0_0_15px_rgba(59,130,246,0)] hover:shadow-[0_0_30px_rgba(59,130,246,0.4)] relative overflow-hidden">
				<div class="flex flex-col items-center justify-center transition-all duration-300">
					<span class="font-display text-3xl font-black text-white group-hover:text-blue-400 transition-colors">5g</span>
					<span class="text-[10px] uppercase font-bold text-zinc-500 tracking-wider mt-1 group-hover:text-blue-400/70">Log Dose</span>
				</div>
			</button>
		`)
	}
}
