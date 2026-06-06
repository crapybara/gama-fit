package handlers

import (
	"fmt"
	"net/http"

	"gama-fit/database"
)

func GetFitnessScore(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, _ := getLocalTime(r)
	score := 30

	currentStreak := GetTrueStreak(userID, localDate)
	if currentStreak >= 3 {
		score += 20
	}

	var weeklyTasks int
	_ = database.DB.QueryRow("SELECT COUNT(*) FROM goals WHERE user_id = $1 AND claimed = 1 AND claimed_at >= TO_CHAR($2::DATE - INTERVAL '7 days', 'YYYY-MM-DD')", userID, localDate).Scan(&weeklyTasks)

	if weeklyTasks >= 50 {
		score += 30
	} else if weeklyTasks >= 25 {
		score += 15
	}

	var weight, height float64
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 ORDER BY log_date DESC LIMIT 1", userID).Scan(&weight)
	_ = database.DB.QueryRow("SELECT height FROM user_stats WHERE user_id = $1", userID).Scan(&height)

	if weight > 0 && height > 0 {
		hMeter := height / 100.0
		bmi := weight / (hMeter * hMeter)
		if bmi >= 18.5 && bmi <= 24.9 {
			score += 15
		}
	}

	if currentStreak > 30 {
		if score < 95 {
			score = 95
		}
	}

	if score > 100 {
		score = 100
	}

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
