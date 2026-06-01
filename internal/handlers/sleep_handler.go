package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"gama-fit/database"
)

func calculateSleepDuration(bedtime, waketime string) (hours int, minutes int, totalMinutes int) {
	layout := "15:04"
	bed, err1 := time.Parse(layout, bedtime)
	wake, err2 := time.Parse(layout, waketime)

	if err1 != nil || err2 != nil {
		return 0, 0, 0
	}

	if wake.Before(bed) {
		wake = wake.Add(24 * time.Hour)
	}

	duration := wake.Sub(bed)
	totalMins := int(duration.Minutes())
	return totalMins / 60, totalMins % 60, totalMins
}

func getQualityScore(quality string, totalMinutes int) int {
	baseScore := 0
	if totalMinutes >= 480 {
		baseScore = 80
	} else if totalMinutes >= 420 {
		baseScore = 70
	} else if totalMinutes >= 360 {
		baseScore = 50
	} else {
		baseScore = 30
	}

	switch quality {
	case "great":
		baseScore += 20
	case "avg":
		baseScore += 5
	case "poor":
		baseScore -= 15
	}

	if baseScore > 100 {
		return 100
	}
	if baseScore < 0 {
		return 0
	}
	return baseScore
}

func HandleSleepSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	var bedtime, waketime, quality string
	var durationMins, score int

	err := database.DB.QueryRow(`
		SELECT bedtime, waketime, quality, duration_mins, score 
		FROM sleep_logs 
		WHERE user_id = $1
		ORDER BY log_date DESC LIMIT 1
	`, userID).Scan(&bedtime, &waketime, &quality, &durationMins, &score)

	if err != nil {
		fmt.Fprintf(w, `
		<div class="absolute top-0 right-0 w-96 h-96 bg-app-sleep/10 blur-3xl rounded-full translate-x-1/2 -translate-y-1/2 pointer-events-none"></div>
		<h3 class="text-white font-black uppercase tracking-wider text-sm mb-6 relative z-10">Last Night's Recovery</h3>
		<div class="text-zinc-500 text-sm text-center py-8 font-mono animate-pulse uppercase tracking-widest relative z-10">No Sleep Logged Yet</div>
		`)
		return
	}

	hrs := durationMins / 60
	mins := durationMins % 60

	qualityText := "Average"
	qualityColor := "text-blue-400"
	if quality == "great" {
		qualityText = "Excellent"
		qualityColor = "text-app-yellow"
	} else if quality == "poor" {
		qualityText = "Poor"
		qualityColor = "text-red-500"
	}

	radius := 45.0
	circumference := 2 * math.Pi * radius
	offset := circumference - (float64(score) / 100.0 * circumference)

	fmt.Fprintf(w, `
		<div class="absolute top-1/2 left-1/4 w-[500px] h-[500px] bg-app-sleep/10 blur-[100px] rounded-full -translate-x-1/2 -translate-y-1/2 pointer-events-none"></div>
		<div class="flex flex-col md:flex-row items-center justify-between gap-8 sm:gap-12 relative z-10">
			<div class="relative flex-shrink-0">
				<div class="absolute inset-0 bg-app-sleep/20 blur-3xl rounded-full scale-90"></div>
				<div class="relative w-48 h-48 sm:w-64 sm:h-64 flex items-center justify-center">
					<svg class="w-full h-full -rotate-90 drop-shadow-[0_0_25px_rgba(99,102,241,0.6)]" viewBox="0 0 100 100">
						<circle class="text-zinc-800/40" stroke-width="5" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50"/>
						<circle class="text-app-sleep sleep-ring" stroke-width="5" stroke-linecap="round" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50" data-target="%d" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f; transition: stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1);"/>
					</svg>
					<div class="absolute inset-0 flex flex-col items-center justify-center text-center">
						<span class="font-display font-black text-5xl sm:text-7xl text-white drop-shadow-lg tracking-tighter">%d</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-app-sleep tracking-widest mt-1 bg-app-sleep/10 px-2 sm:px-3 py-1 rounded-full border border-app-sleep/20 shadow-[0_0_10px_rgba(99,102,241,0.2)]">Score</span>
					</div>
				</div>
			</div>

			<div class="flex-1 w-full grid grid-cols-2 gap-3 sm:gap-4">
				<div class="bg-zinc-900/50 border border-white/5 rounded-[1.5rem] p-4 sm:p-6 hover:bg-zinc-900/80 hover:border-app-sleep/30 transition-all duration-300 group">
					<div class="flex items-center gap-2 mb-2 sm:mb-3">
						<svg class="w-4 h-4 sm:w-5 sm:h-5 text-zinc-500 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
						<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest text-nowrap">Time Asleep</span>
					</div>
					<span class="font-display font-black text-2xl sm:text-4xl text-white">%d<span class="text-lg sm:text-xl text-zinc-500 font-bold">h</span> %d<span class="text-lg sm:text-xl text-zinc-500 font-bold">m</span></span>
				</div>

				<div class="bg-zinc-900/50 border border-white/5 rounded-[1.5rem] p-4 sm:p-6 hover:bg-zinc-900/80 transition-all duration-300 group">
					<div class="flex items-center gap-2 mb-2 sm:mb-3">
						<svg class="w-4 h-4 sm:w-5 sm:h-5 text-zinc-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
						<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest text-nowrap">Quality</span>
					</div>
					<span class="font-display font-black text-xl sm:text-3xl %s">%s</span>
				</div>

				<div class="bg-zinc-900/50 border border-white/5 rounded-[1.5rem] p-4 sm:p-6 hover:bg-zinc-900/80 hover:border-blue-400/30 transition-all duration-300 group">
					<div class="flex items-center gap-2 mb-2 sm:mb-3">
						<svg class="w-4 h-4 sm:w-5 sm:h-5 text-zinc-500 group-hover:text-blue-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path></svg>
						<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest text-nowrap">Bedtime</span>
					</div>
					<span class="font-mono font-bold text-xl sm:text-3xl text-zinc-300">%s</span>
				</div>

				<div class="bg-zinc-900/50 border border-white/5 rounded-[1.5rem] p-4 sm:p-6 hover:bg-zinc-900/80 hover:border-orange-400/30 transition-all duration-300 group">
					<div class="flex items-center gap-2 mb-2 sm:mb-3">
						<svg class="w-4 h-4 sm:w-5 sm:h-5 text-zinc-500 group-hover:text-orange-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
						<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest text-nowrap">Wake up</span>
					</div>
					<span class="font-mono font-bold text-xl sm:text-3xl text-zinc-300">%s</span>
				</div>
			</div>
		</div>
	`, score, circumference, offset, score, hrs, mins, qualityColor, qualityText, formatTime12h(bedtime), formatTime12h(waketime))
}

func HandleSleep(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, _ := getLocalTime(r)
	if r.Method == http.MethodPost {
		bedtime := r.FormValue("bedtime")
		waketime := r.FormValue("waketime")
		quality := r.FormValue("quality")

		_, _, durationMins := calculateSleepDuration(bedtime, waketime)
		score := getQualityScore(quality, durationMins)

		_, _ = database.DB.Exec(`
			INSERT INTO sleep_logs (user_id, log_date, bedtime, waketime, quality, duration_mins, score) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT(user_id, log_date) DO UPDATE SET 
			bedtime = excluded.bedtime, waketime = excluded.waketime, quality = excluded.quality, duration_mins = excluded.duration_mins, score = excluded.score
		`, userID, localDate, bedtime, waketime, quality, durationMins, score)

		fmt.Fprint(w, `<div id="sleep-summary" hx-swap-oob="true" hx-get="/api/sleep/summary" hx-trigger="load" class="glass-panel rounded-[2.5rem] p-8 lg:p-12 relative overflow-hidden page-animate-fade layout-delay"></div>`)
	} else if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		_, _ = database.DB.Exec("DELETE FROM sleep_logs WHERE id = $1 AND user_id = $2", id, userID)
		fmt.Fprint(w, `<div id="sleep-summary" hx-swap-oob="true" hx-get="/api/sleep/summary" hx-trigger="load" class="glass-panel rounded-[2.5rem] p-8 lg:p-12 relative overflow-hidden page-animate-fade layout-delay"></div>`)
	}

	HandleSleepHistory(w, r)
}

func HandleSleepHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	rows, err := database.DB.Query(`
		SELECT id, log_date, bedtime, waketime, quality, duration_mins 
		FROM sleep_logs 
		WHERE user_id = $1
		ORDER BY log_date DESC 
		LIMIT 14
	`, userID)
	if err != nil {
		fmt.Fprint(w, `<div class="text-zinc-500 text-sm py-4 font-mono col-span-full text-center">No sleep history recorded.</div>`)
		return
	}
	defer rows.Close()

	hasHistory := false
	html := ""
	for rows.Next() {
		hasHistory = true
		var id, durationMins int
		var day, bed, wake, quality string
		if err := rows.Scan(&id, &day, &bed, &wake, &quality, &durationMins); err == nil {
			hrs := durationMins / 60
			mins := durationMins % 60

			qualityText := "Average"
			qualityColor := "text-blue-400"
			if quality == "great" {
				qualityText = "Great"
				qualityColor = "text-app-yellow"
			} else if quality == "poor" {
				qualityText = "Poor"
				qualityColor = "text-red-500"
			}

			html += fmt.Sprintf(`
		<div class="bg-zinc-900/30 border border-zinc-800/50 rounded-2xl p-5 hover:border-app-sleep/50 transition-colors group relative">
			<div class="flex justify-between items-start mb-4">
				<div>
					<span class="text-white font-bold text-sm block">%s</span>
					<span class="%s text-[10px] font-bold uppercase tracking-widest">%s</span>
				</div>
				<span class="font-display font-black text-2xl text-white">%d<span class="text-sm text-zinc-500">h</span> %d<span class="text-sm text-zinc-500">m</span></span>
			</div>
			<div class="flex justify-between text-zinc-500 font-mono text-xs">
				<span>%s</span>
				<span>%s</span>
			</div>
			<button hx-delete="/api/sleep?id=%d" hx-target="#sleep-history-list" hx-swap="innerHTML" class="absolute -top-2 -right-2 bg-red-500/20 text-red-500 hover:bg-red-500 hover:text-white rounded-full p-1.5 opacity-0 group-hover:opacity-100 transition-all scale-75 group-hover:scale-100">
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
			</button>
		</div>`, dayLabel(day), qualityColor, qualityText, hrs, mins, formatTime12h(bed), formatTime12h(wake), id)
		}
	}

	if !hasHistory {
		html = `<div class="text-zinc-500 text-sm py-4 font-mono col-span-full text-center">No sleep history recorded.</div>`
	}

	fmt.Fprint(w, html)
}
