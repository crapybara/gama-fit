package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gama-fit/database"
)

func HandleCheckins(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	targetYear := now.Year()
	targetMonth := now.Month()

	if y := r.URL.Query().Get("year"); y != "" {
		if parsedY, err := strconv.Atoi(y); err == nil {
			targetYear = parsedY
		}
	}
	if m := r.URL.Query().Get("month"); m != "" {
		if parsedM, err := strconv.Atoi(m); err == nil {
			targetMonth = time.Month(parsedM)
		}
	}

	firstOfMonth := time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	prevMonth := firstOfMonth.AddDate(0, -1, 0)
	nextMonth := firstOfMonth.AddDate(0, 1, 0)

	if r.Method == "POST" {
		toggleDate := r.URL.Query().Get("date")
		if toggleDate == "" {
			toggleDate = now.Format("2006-01-02")
		}

		var count int
		_ = database.DB.QueryRow("SELECT COUNT(*) FROM checkins WHERE checkin_date = ?", toggleDate).Scan(&count)
		if count > 0 {
			_, _ = database.DB.Exec("DELETE FROM checkins WHERE checkin_date = ?", toggleDate)
		} else {
			_, _ = database.DB.Exec("INSERT INTO checkins (checkin_date) VALUES (?)", toggleDate)
		}

		rows, err := database.DB.Query("SELECT checkin_date FROM checkins")
		if err == nil {
			allCheckins := make(map[string]bool)
			for rows.Next() {
				var d string
				if err := rows.Scan(&d); err == nil {
					allCheckins[d] = true
				}
			}
			rows.Close()

			realStreak := 0
			todayStr := now.Format("2006-01-02")
			yesterdayStr := now.AddDate(0, 0, -1).Format("2006-01-02")

			if allCheckins[todayStr] || allCheckins[yesterdayStr] {
				checkDate := now
				if !allCheckins[todayStr] {
					checkDate = now.AddDate(0, 0, -1)
				}
				for {
					if allCheckins[checkDate.Format("2006-01-02")] {
						realStreak++
						checkDate = checkDate.AddDate(0, 0, -1)
					} else {
						break
					}
				}
			}
			_, _ = database.DB.Exec("UPDATE user_stats SET current_streak = ? WHERE id = 1", realStreak)
		}
	}

	rows, err := database.DB.Query("SELECT checkin_date FROM checkins WHERE checkin_date >= ? AND checkin_date <= ?", firstOfMonth.Format("2006-01-02"), lastOfMonth.Format("2006-01-02"))
	if err != nil {
		w.Write([]byte(`<div class="text-red-500">Database Error</div>`))
		return
	}
	defer rows.Close()

	checkinMap := make(map[string]bool)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			checkinMap[d] = true
		}
	}

	html := `<div class="w-full">`
	html += `<div class="flex flex-col sm:flex-row items-center justify-between mb-6 gap-4">`
	html += `<div class="flex items-center gap-2 bg-zinc-900/80 p-1.5 rounded-xl border border-zinc-800">`
	html += fmt.Sprintf(`<button hx-get="/api/checkins?year=%d&month=%d" hx-target="#checkin-board-container" class="p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-white transition-colors"><svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg></button>`, prevMonth.Year(), prevMonth.Month())
	html += fmt.Sprintf(`<h4 class="text-white font-black text-lg tracking-tight w-28 text-center uppercase">%s %d</h4>`, targetMonth.String()[:3], targetYear)
	html += fmt.Sprintf(`<button hx-get="/api/checkins?year=%d&month=%d" hx-target="#checkin-board-container" class="p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-white transition-colors"><svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg></button>`, nextMonth.Year(), nextMonth.Month())
	html += `</div>`

	if targetMonth == now.Month() && targetYear == now.Year() {
		if checkinMap[now.Format("2006-01-02")] {
			html += `<button disabled class="bg-blue-500/20 text-blue-400 font-bold px-6 py-2.5 rounded-xl cursor-not-allowed border border-blue-500/50 w-full sm:w-auto">✅ Checked In Today</button>`
		} else {
			html += fmt.Sprintf(`<button hx-post="/api/checkins?date=%s&year=%d&month=%d" hx-target="#checkin-board-container" class="bg-blue-600 hover:bg-blue-500 text-white font-bold px-6 py-2.5 rounded-xl transition-all shadow-[0_0_15px_rgba(59,130,246,0.4)] hover:scale-105 w-full sm:w-auto">Check In Today</button>`, now.Format("2006-01-02"), targetYear, targetMonth)
		}
	}
	html += `</div>`

	html += `<div class="grid grid-cols-7 gap-2 text-center text-[10px] font-bold text-zinc-600 mb-2 uppercase tracking-wider">`
	html += `<div>Sun</div><div>Mon</div><div>Tue</div><div>Wed</div><div>Thu</div><div>Fri</div><div>Sat</div></div>`
	html += `<div class="grid grid-cols-7 gap-2 sm:gap-3">`

	startWeekday := int(firstOfMonth.Weekday())
	for i := 0; i < startWeekday; i++ {
		html += `<div></div>`
	}

	for i := 1; i <= lastOfMonth.Day(); i++ {
		currentDate := time.Date(targetYear, targetMonth, i, 0, 0, 0, 0, time.Local)
		dateStr := currentDate.Format("2006-01-02")
		isActive := checkinMap[dateStr]
		isFuture := currentDate.After(now)

		classes := "h-10 sm:h-12 rounded-lg sm:rounded-xl flex items-center justify-center font-bold text-sm transition-all "

		if isFuture {
			classes += "bg-zinc-900/20 text-zinc-700 border border-white/5 cursor-not-allowed"
			html += fmt.Sprintf(`<div class="%s">%d</div>`, classes, i)
		} else {
			if isActive {
				classes += "bg-blue-500 text-white shadow-[0_0_10px_rgba(59,130,246,0.4)] border border-blue-400 hover:bg-red-500 hover:border-red-400 hover:shadow-[0_0_15px_rgba(239,68,68,0.4)] cursor-pointer"
			} else {
				classes += "bg-zinc-800/40 text-zinc-500 border border-white/5 hover:border-blue-500 hover:text-blue-400 cursor-pointer"
			}

			if i == now.Day() && targetMonth == now.Month() && targetYear == now.Year() && !isActive {
				classes += " ring-2 ring-blue-500/50 text-white bg-zinc-800"
			}

			html += fmt.Sprintf(`<button hx-post="/api/checkins?date=%s&year=%d&month=%d" hx-target="#checkin-board-container" class="%s">%d</button>`, dateStr, targetYear, targetMonth, classes, i)
		}
	}
	html += `</div></div>`
	html += `</div></div>`

	if r.Method == "POST" {
		w.Header().Set("HX-Trigger", "updateRings")
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
