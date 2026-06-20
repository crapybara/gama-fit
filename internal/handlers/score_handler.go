package handlers

import (
	"fmt"
	"math"
	"net/http"

	"gama-fit/database"
)

func GetFitnessScore(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, _ := getLocalTime(r)
	currentStreak := GetTrueStreak(userID, localDate)

	var weight, height float64
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 ORDER BY log_date DESC LIMIT 1", userID).Scan(&weight)
	_ = database.DB.QueryRow("SELECT height FROM user_stats WHERE user_id = $1", userID).Scan(&height)

	// 1. Sleep (25 pts)
	sleepScore := float64(calculate14DaySleepScore(userID))
	ptsSleep := (sleepScore / 100.0) * 25.0

	// 2. Body Comp (25 pts)
	// Using BMI, optimal ~22. Bell curve decay.
	ptsBody := 0.0
	if weight > 0 && height > 0 {
		hMeter := height / 100.0
		bmi := weight / (hMeter * hMeter)
		diff := math.Abs(bmi - 22.0)
		ptsBody = 25.0 * math.Exp(-0.5*math.Pow(diff/4.0, 2))
	} else {
		ptsBody = 15.0 // fallback
	}

	// 3. Activity (25 pts)
	var activeDays int
	_ = database.DB.QueryRow("SELECT COUNT(DISTINCT logged_date) FROM freestyle_logs WHERE user_id = $1 AND logged_date >= TO_CHAR($2::DATE - INTERVAL '14 days', 'YYYY-MM-DD')", userID, localDate).Scan(&activeDays)
	ptsActivity := (float64(activeDays) / 8.0) * 25.0 // optimal ~8 days / 14 days
	if ptsActivity > 25.0 {
		ptsActivity = 25.0
	}

	// 4. Consistency (25 pts)
	ptsStreak := math.Min(25.0, (float64(currentStreak)/14.0)*25.0)

	total := int(math.Round(ptsSleep + ptsBody + ptsActivity + ptsStreak))
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}
	score := total

	html := fmt.Sprintf(`
		<span class="font-display text-5xl font-black text-white">%d</span>
		<script>
			setTimeout(() => {
				if(typeof window.animateRing === 'function') {
					window.animateRing('scoreRing', %d);
				}
			}, 50);
		</script>
	`, score, score)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
