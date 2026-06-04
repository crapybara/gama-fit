package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gama-fit/database"
)

// --- 1. FREESTYLE TRACKER ---
func HandleFreestyle(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	if r.URL.Query().Get("action") == "options" {
		rows, err := database.DB.Query("SELECT DISTINCT exercise_name FROM workout_plans WHERE user_id = $1 AND exercise_name != '' ORDER BY exercise_name ASC", userID)
		if err != nil {
			log.Printf("Error fetching exercise options: %v", err)
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

	if r.URL.Query().Get("action") == "plan_options" {
		dayStr := r.URL.Query().Get("day")
		if dayStr == "" {
			dayStr = "1"
		}
		day, _ := strconv.Atoi(dayStr)

		rows, err := database.DB.Query("SELECT exercise_name FROM workout_plans WHERE user_id = $1 AND day_of_week = $2 ORDER BY id ASC", userID, day)
		if err != nil {
			log.Printf("Error fetching plan options: %v", err)
			w.Write([]byte("<option value=\"\">Error loading plan</option>"))
			return
		}
		defer rows.Close()

		html := `<option value="">Quick Select...</option>`
		count := 1
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				html += fmt.Sprintf(`<option value="%s">m%d - %s</option>`, name, count, name)
				count++
			}
		}
		w.Write([]byte(html))
		return
	}

	if r.Method == http.MethodPost {
		localDate, localTime := getLocalTime(r)
		exercise := r.FormValue("exercise")
		weightStr := r.FormValue("weight")
		repsStr := r.FormValue("reps")

		if exercise != "" && weightStr != "" && repsStr != "" {
			weight, err1 := strconv.ParseFloat(weightStr, 64)
			reps, err2 := strconv.Atoi(repsStr)
			if err1 == nil && err2 == nil {
				// We also fetch muscle from workout_plans if available
				var muscle string
				database.DB.QueryRow("SELECT muscle FROM workout_plans WHERE user_id = $1 AND exercise_name = $2 LIMIT 1", userID, exercise).Scan(&muscle)

				_, err := database.DB.Exec("INSERT INTO freestyle_logs (user_id, exercise_name, weight, reps, sets, muscle, logged_date, logged_time) VALUES ($1, $2, $3, $4, 1, $5, $6, $7)", userID, exercise, weight, reps, muscle, localDate, localTime)
				if err != nil {
					log.Printf("Error inserting freestyle log: %v", err)
				}
			}
		}
	} else if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		_, err := database.DB.Exec("DELETE FROM freestyle_logs WHERE id = $1 AND user_id = $2", id, userID)
		if err != nil {
			log.Printf("Error deleting freestyle log: %v", err)
		}
	}

	localDate, _ := getLocalTime(r)
	rows, err := database.DB.Query("SELECT id, exercise_name, weight, reps FROM freestyle_logs WHERE user_id = $1 AND logged_date = $2 ORDER BY id DESC", userID, localDate)
	if err != nil {
		log.Printf("Error fetching freestyle logs: %v", err)
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
				<button hx-delete="/api/freestyle?id=%d" hx-target="#freestyle-list" hx-confirm="Delete this set?" class="ml-2 text-zinc-600 hover:text-red-500"><svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
			</div>
		</div>`, name, weight, reps, id)
		}
	}

	if html == "" {
		html = `<div class="text-zinc-500 text-xs font-mono text-center py-4 border-2 border-dashed border-zinc-800/50 rounded-2xl uppercase tracking-widest">No sets recorded for today</div>`
	}

	w.Write([]byte(html))
}

func HandleMarkLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _ := GetUserID(r)
	localDate, localTime := getLocalTime(r)
	content := r.FormValue("content")

	if content == "" {
		return
	}

	dayMap := map[string]int{
		"monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4, "friday": 5, "saturday": 6, "sunday": 7,
		"mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6, "sun": 7,
	}

	// Helper to resolve aliases for a specific day
	getAliases := func(day int) map[string]string {
		aliases := make(map[string]string)
		rows, err := database.DB.Query("SELECT exercise_name FROM workout_plans WHERE user_id = $1 AND day_of_week = $2 ORDER BY id ASC", userID, day)
		if err == nil {
			defer rows.Close()
			count := 1
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err == nil {
					aliases[fmt.Sprintf("m%d", count)] = name
					count++
				}
			}
		}
		return aliases
	}

	lines := strings.Split(content, "\n")
	
	// Default to today's day of week
	parsedDate, _ := time.Parse("2006-01-02", localDate)
	currentDayOfWeek := int(parsedDate.Weekday())
	if currentDayOfWeek == 0 {
		currentDayOfWeek = 7
	}
	
	targetDayOfWeek := currentDayOfWeek
	targetDate := localDate
	var aliases map[string]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 1. Check for day header (#monday)
		if strings.HasPrefix(line, "#") {
			dayName := strings.ToLower(strings.TrimPrefix(line, "#"))
			if d, ok := dayMap[dayName]; ok {
				targetDayOfWeek = d
				// Calculate targetDate relative to today (localDate)
				diff := targetDayOfWeek - currentDayOfWeek
				if diff > 0 {
					diff -= 7 // It was in the past
				}
				targetDate = parsedDate.AddDate(0, 0, diff).Format("2006-01-02")
				aliases = getAliases(targetDayOfWeek)
			}
			continue
		}

		// Lazy load aliases if not set by header
		if aliases == nil {
			aliases = getAliases(targetDayOfWeek)
		}

		// 2. Parse exercise sets: m1: 10x60, 8x70
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		exerciseAlias := strings.ToLower(strings.TrimSpace(parts[0]))
		exerciseName, ok := aliases[exerciseAlias]
		if !ok {
			// Maybe it's not an alias but a direct name?
			exerciseName = parts[0]
		}

		setsContent := parts[1]
		sets := strings.Split(setsContent, ",")

		for _, set := range sets {
			set = strings.TrimSpace(set)
			if set == "" {
				continue
			}

			// Format: 10x60 (reps x weight)
			setParts := strings.Split(strings.ToLower(set), "x")
			if len(setParts) != 2 {
				continue
			}

			reps, _ := strconv.Atoi(strings.TrimSpace(setParts[0]))
			weight, _ := strconv.ParseFloat(strings.TrimSpace(setParts[1]), 64)

			if reps > 0 && weight > 0 {
				var muscle string
				database.DB.QueryRow("SELECT muscle FROM workout_plans WHERE user_id = $1 AND exercise_name = $2 LIMIT 1", userID, exerciseName).Scan(&muscle)

				_, err := database.DB.Exec("INSERT INTO freestyle_logs (user_id, exercise_name, weight, reps, sets, muscle, logged_date, logged_time) VALUES ($1, $2, $3, $4, 1, $5, $6, $7)", userID, exerciseName, weight, reps, muscle, targetDate, localTime)
				if err != nil {
					log.Printf("Error inserting marklog entry: %v", err)
				}
			}
		}
	}

	// Trigger HTMX refresh
	http.Redirect(w, r, "/api/freestyle?local_date="+localDate, http.StatusSeeOther)
}


// --- 2. WORKOUT PLAN (7-Day System) ---
func HandleWorkoutPlan(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	dayStr := r.URL.Query().Get("day")
	if dayStr == "" {
		dayStr = "1"
	}
	day, _ := strconv.Atoi(dayStr)

	if r.Method == "POST" {
		var count int
		err := database.DB.QueryRow("SELECT COUNT(*) FROM workout_plans WHERE user_id = $1 AND day_of_week = $2", userID, day).Scan(&count)
		if err != nil {
			log.Printf("Error counting workout plans: %v", err)
		}

		if count < 15 {
			exercise := r.FormValue("exercise")
			setsStr := r.FormValue("sets")
			reps := r.FormValue("reps")
			muscle := r.FormValue("muscle")

			sets, err := strconv.Atoi(setsStr)
			if err != nil || sets <= 0 {
				sets = 3
			}

			if exercise != "" {
				_, err = database.DB.Exec("INSERT INTO workout_plans (user_id, day_of_week, exercise_name, sets, reps, muscle) VALUES ($1, $2, $3, $4, $5, $6)", userID, day, exercise, sets, reps, muscle)
				if err != nil {
					log.Printf("Error inserting workout plan: %v", err)
				}
			}
		}
	} else if r.Method == "DELETE" {
		id := r.URL.Query().Get("id")
		_, err := database.DB.Exec("DELETE FROM workout_plans WHERE id = $1 AND user_id = $2", id, userID)
		if err != nil {
			log.Printf("Error deleting workout plan: %v", err)
		}
	}

	rows, err := database.DB.Query("SELECT id, exercise_name, sets, reps, muscle FROM workout_plans WHERE user_id = $1 AND day_of_week = $2 ORDER BY id ASC", userID, day)
	if err != nil {
		log.Printf("Error fetching workout plans: %v", err)
		w.Write([]byte(`<div class="text-zinc-500 text-sm text-center py-6">Rest day or no exercises added yet.</div>`))
		return
	}
	defer rows.Close()

	exerciseCount := 0
	html := `<div class="space-y-2 mt-4">`
	for rows.Next() {
		exerciseCount++
		var id int
		var name, reps, msl string
		var sets int
		if err := rows.Scan(&id, &name, &sets, &reps, &msl); err == nil {
			html += fmt.Sprintf(`
		<div class="flex items-center justify-between bg-zinc-900/40 border border-white/5 rounded-xl p-3 hover:bg-zinc-900/80 transition-colors">
			<span class="text-zinc-300 text-sm font-medium">%s <span class="text-[9px] text-zinc-500 uppercase tracking-widest ml-2">%s</span></span>
			<div class="flex items-center gap-2 sm:gap-3">
				<span class="text-[10px] font-bold uppercase tracking-wider text-blue-400 bg-blue-400/10 px-2 py-1 rounded">%d Sets</span>
				<span class="text-[10px] font-bold uppercase tracking-wider text-zinc-500 bg-zinc-800 px-2 py-1 rounded">%s Reps</span>
				<button hx-delete="/api/plans?day=%d&id=%d" hx-target="#plans-container" class="text-zinc-600 hover:text-red-500 transition-colors"><svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
			</div>
		</div>`, name, msl, sets, reps, day, id)
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
		<form hx-post="/api/plans?day=%d" hx-target="#plans-container" hx-on::after-request="this.reset()" class="flex flex-col gap-3 mt-6">
			<div class="flex flex-col sm:flex-row gap-3">
				<input type="text" name="exercise" list="exercise-list" placeholder="Exercise name..." autocomplete="off" required class="flex-[3] bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-pink transition-colors">
				<div class="flex gap-2 flex-1">
					<input type="number" name="sets" placeholder="Sets" value="3" min="1" max="10" required class="flex-1 min-w-0 bg-zinc-900/50 border border-zinc-700 rounded-xl px-2 py-3 text-sm text-center text-white outline-none focus:border-app-pink transition-colors font-mono">
					<input type="text" name="reps" placeholder="Reps" value="8-10" required class="flex-1 min-w-0 bg-zinc-900/50 border border-zinc-700 rounded-xl px-2 py-3 text-sm text-center text-white outline-none focus:border-app-pink transition-colors font-mono">
				</div>
			</div>
			<div class="flex gap-3">
				<select name="muscle" required class="flex-1 bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-xs text-zinc-400 outline-none focus:border-app-pink transition-colors appearance-none font-bold">
					<option value="" disabled selected>Muscle Group</option>
					<option value="chest">Chest</option>
					<option value="back">Back</option>
					<option value="shoulders">Shoulders</option>
					<option value="biceps">Biceps</option>
					<option value="triceps">Triceps</option>
					<option value="forearms">Forearms</option>
					<option value="quads">Quads</option>
					<option value="hamstrings">Hamstrings</option>
					<option value="glutes">Glutes</option>
					<option value="calves">Calves</option>
					<option value="abs">Abs</option>
					<option value="obliques">Obliques/Hips</option>
					<option value="erectors">Erectors (Lower Back)</option>
					<option value="traps">Traps</option>
					<option value="neck">Neck</option>
					<option value="cardio">Cardio</option>
				</select>
				<button type="submit" class="bg-app-pink text-white font-bold px-8 rounded-xl hover:bg-pink-500 transition-all shadow-[0_0_15px_rgba(255,0,160,0.2)] flex items-center justify-center">
					<svg class="w-5 h-5 mr-2" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
					ADD EXERCISE
				</button>
			</div>
		</form>`, day)
	}

	w.Write([]byte(html + formHtml))
}

type MuscleStats struct {
	Volume    float64 `json:"volume"`
	Sets      int     `json:"sets"`
	Exercises int     `json:"exercises"`
	Change    float64 `json:"change"`
}

func HandleWorkoutHeatmap(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	scope := r.URL.Query().Get("scope")
	dayStr := r.URL.Query().Get("day")

	repsCalc := `(
		CASE 
			WHEN reps LIKE '%-%' THEN (CAST(split_part(reps, '-', 1) AS DOUBLE PRECISION) + CAST(split_part(reps, '-', 2) AS DOUBLE PRECISION)) / 2.0
			WHEN reps ~ '^[0-9]+$' THEN CAST(reps AS DOUBLE PRECISION)
			ELSE 0
		END
	)`

	// 1. Get Weekly Stats for global normalization and defaults
	weeklyQuery := fmt.Sprintf("SELECT muscle, SUM(sets * %s), SUM(sets), COUNT(*) FROM workout_plans WHERE user_id = $1 GROUP BY muscle", repsCalc)
	rowsW, err := database.DB.Query(weeklyQuery, userID)
	if err != nil {
		log.Printf("Weekly heatmap query error: %v", err)
	}
	defer rowsW.Close()

	weeklyStats := make(map[string]MuscleStats)
	weeklyMax := 0.0
	totalWeekly := MuscleStats{}

	for rowsW.Next() {
		var muscle string
		var vol, sts float64 // Scan sets as float to handle SUM then cast
		var ex int
		if err := rowsW.Scan(&muscle, &vol, &sts, &ex); err == nil && muscle != "" {
			stats := MuscleStats{Volume: vol, Sets: int(sts), Exercises: ex}
			weeklyStats[muscle] = stats
			if vol > weeklyMax {
				weeklyMax = vol
			}
			totalWeekly.Volume += vol
			totalWeekly.Sets += int(sts)
			totalWeekly.Exercises += ex
		}
	}

	// 2. Get Scope-specific Stats
	heatmap := make(map[string]MuscleStats)
	totalScoped := MuscleStats{}

	if scope == "day" && dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil {
			dayQuery := fmt.Sprintf("SELECT muscle, SUM(sets * %s), SUM(sets), COUNT(*) FROM workout_plans WHERE user_id = $1 AND day_of_week = $2 GROUP BY muscle", repsCalc)
			rowsD, err := database.DB.Query(dayQuery, userID, day)
			if err == nil {
				defer rowsD.Close()
				for rowsD.Next() {
					var m string
					var v, s float64
					var e int
					if err := rowsD.Scan(&m, &v, &s, &e); err == nil && m != "" {
						st := MuscleStats{Volume: v, Sets: int(s), Exercises: e}
						heatmap[m] = st
						totalScoped.Volume += v
						totalScoped.Sets += int(s)
						totalScoped.Exercises += e
					}
				}
			}
		}
	} else {
		heatmap = weeklyStats
		totalScoped = totalWeekly
	}

	// 3. Return structured response
	response := struct {
		Muscles   map[string]MuscleStats `json:"muscles"`
		Total     MuscleStats            `json:"total"`
		WeeklyMax float64                `json:"weeklyMax"`
	}{
		Muscles:   heatmap,
		Total:     totalScoped,
		WeeklyMax: weeklyMax,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --- 3. BODY WEIGHT TRACKER ---
func HandleBodyWeight(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, _ := getLocalTime(r)
	if r.Method == http.MethodPost {
		weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
		if weight > 0 {
			_, _ = database.DB.Exec("INSERT INTO body_weight_logs (user_id, weight, log_date) VALUES ($1, $2, $3) ON CONFLICT(user_id, log_date) DO UPDATE SET weight=excluded.weight", userID, weight, localDate)
		}
	} else if r.Method == http.MethodDelete {
		_, _ = database.DB.Exec("DELETE FROM body_weight_logs WHERE user_id = $1 AND log_date = $2", userID, localDate)
	}

	var todayWeight float64
	err := database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 AND log_date = $2", userID, localDate).Scan(&todayWeight)

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

// --- 4. CARDIO TRACKER ---
func HandleCardio(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, localTime := getLocalTime(r)

	if r.Method == http.MethodPost {
		hr, _ := strconv.Atoi(r.FormValue("heart_rate"))
		dur, _ := strconv.Atoi(r.FormValue("duration"))
		pace := r.FormValue("pace")
		intensity := r.FormValue("intensity")

		if hr > 0 && dur > 0 {
			_, err := database.DB.Exec("INSERT INTO cardio_logs (user_id, heart_rate, duration, pace, intensity, logged_date, logged_time) VALUES ($1, $2, $3, $4, $5, $6, $7)", userID, hr, dur, pace, intensity, localDate, localTime)
			if err != nil {
				log.Printf("Error inserting cardio log: %v", err)
			}
		}
	} else if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		_, err := database.DB.Exec("DELETE FROM cardio_logs WHERE id = $1 AND user_id = $2", id, userID)
		if err != nil {
			log.Printf("Error deleting cardio log: %v", err)
		}
	}

	var age int
	_ = database.DB.QueryRow("SELECT age FROM user_stats WHERE user_id = $1", userID).Scan(&age)
	if age <= 0 {
		age = 25
	}
	mhr := 220 - age

	rows, err := database.DB.Query("SELECT id, heart_rate, duration, pace, intensity FROM cardio_logs WHERE user_id = $1 AND logged_date = $2 ORDER BY id DESC", userID, localDate)
	if err != nil {
		log.Printf("Error fetching cardio logs: %v", err)
	}
	defer rows.Close()

	logsHtml := ""
	for rows.Next() {
		var id, hr, dur int
		var pace, intensity string
		if err := rows.Scan(&id, &hr, &dur, &pace, &intensity); err == nil {
			logsHtml += fmt.Sprintf(`
			<div class="flex items-center justify-between bg-zinc-900/40 border border-white/5 rounded-xl p-3 mb-2 hover:bg-zinc-900/80 transition-colors">
				<div class="flex flex-col">
					<span class="text-white text-sm font-bold">%s</span>
					<span class="text-[10px] text-zinc-500 uppercase tracking-tighter">%s</span>
				</div>
				<div class="flex items-center gap-3 text-xs font-mono">
					<span class="text-blue-400">%d BPM</span>
					<span class="text-zinc-600">|</span>
					<span class="text-emerald-400">%d MIN</span>
					<button hx-delete="/api/cardio?id=%d" hx-target="#cardio-list" hx-confirm="Delete this entry?" class="ml-2 text-zinc-600 hover:text-red-500 transition-colors">
						<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
					</button>
				</div>
			</div>`, pace, intensity, hr, dur, id)
		}
	}

	if logsHtml == "" {
		logsHtml = `<div class="text-zinc-600 text-[10px] font-mono text-center py-4 border border-dashed border-zinc-800/50 rounded-xl uppercase tracking-widest">No cardio sessions yet</div>`
	}

	zonesHtml := fmt.Sprintf(`
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-2 mb-6">
			<div class="bg-emerald-500/5 border border-emerald-500/10 rounded-xl p-2.5 text-center">
				<p class="text-[9px] font-black text-emerald-500 uppercase tracking-tighter">Light (50-60%%)</p>
				<p class="text-xs font-bold text-white mt-0.5">%d-%d</p>
				<p class="text-[8px] text-zinc-500 leading-tight mt-1">Slow walking</p>
			</div>
			<div class="bg-blue-500/5 border border-blue-500/10 rounded-xl p-2.5 text-center">
				<p class="text-[9px] font-black text-blue-400 uppercase tracking-tighter">Mod (60-75%%)</p>
				<p class="text-xs font-bold text-white mt-0.5">%d-%d</p>
				<p class="text-[8px] text-zinc-500 leading-tight mt-1">Brisk walking</p>
			</div>
			<div class="bg-app-yellow/5 border border-app-yellow/10 rounded-xl p-2.5 text-center">
				<p class="text-[9px] font-black text-app-yellow uppercase tracking-tighter">Vigorous (75-85%%)</p>
				<p class="text-xs font-bold text-white mt-0.5">%d-%d</p>
				<p class="text-[8px] text-zinc-500 leading-tight mt-1">Running/Cycling</p>
			</div>
			<div class="bg-app-pink/5 border border-app-pink/10 rounded-xl p-2.5 text-center">
				<p class="text-[9px] font-black text-app-pink uppercase tracking-tighter">Hard (85-95%%+)</p>
				<p class="text-xs font-bold text-white mt-0.5">%d-%d</p>
				<p class="text-[8px] text-zinc-500 leading-tight mt-1">Sprints/HIIT</p>
			</div>
		</div>
	`, 
	int(float64(mhr)*0.5), int(float64(mhr)*0.6),
	int(float64(mhr)*0.6), int(float64(mhr)*0.75),
	int(float64(mhr)*0.75), int(float64(mhr)*0.85),
	int(float64(mhr)*0.85), int(float64(mhr)*0.95))

	formHtml := `
		<form hx-post="/api/cardio" hx-target="#cardio-list" hx-on::after-request="this.reset()" class="flex flex-col gap-3 mb-6">
			<div class="flex flex-col sm:flex-row gap-3">
				<div class="flex-1">
					<input type="number" name="heart_rate" placeholder="HR (BPM)" required class="w-full bg-zinc-900/60 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 transition-all font-mono">
				</div>
				<div class="flex-1">
					<input type="number" name="duration" placeholder="DUR (MIN)" required class="w-full bg-zinc-900/60 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-emerald-500 transition-all font-mono">
				</div>
				<div class="flex-[1.5]">
					<input type="text" name="pace" placeholder="Pace (e.g. 5:30 min/km)" required class="w-full bg-zinc-900/60 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-yellow transition-all font-bold">
				</div>
			</div>
			<div class="flex gap-3">
				<div class="relative flex-1">
					<select name="intensity" class="w-full bg-zinc-900/60 border border-zinc-800 rounded-xl px-4 py-3 text-sm text-zinc-400 outline-none focus:border-app-pink transition-all appearance-none cursor-pointer">
						<option value="Light">Light Intensity</option>
						<option value="Moderate" selected>Moderate Intensity</option>
						<option value="Vigorous">Vigorous Intensity</option>
						<option value="Very Hard">Very Hard / HIIT</option>
					</select>
					<div class="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
						<svg class="w-4 h-4 text-app-pink" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
					</div>
				</div>
				<button type="submit" class="bg-white text-black font-black px-6 rounded-xl hover:bg-zinc-200 transition-all shadow-xl flex items-center justify-center shrink-0">
					<svg class="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
				</button>
			</div>
		</form>`

	if r.Header.Get("HX-Request") == "true" && r.Method == http.MethodPost {
		w.Write([]byte(logsHtml))
	} else if r.Header.Get("HX-Request") == "true" && r.Method == http.MethodDelete {
		w.Write([]byte(logsHtml))
	} else {
		w.Write([]byte(zonesHtml + formHtml + `<div id="cardio-list">` + logsHtml + `</div>`))
	}
}
