package handlers

import (
	"fmt"
	"net/http"

	"gama-fit/database"
)

func GetFitnessScore(w http.ResponseWriter, r *http.Request) {
	score := 30

	currentStreak := GetTrueStreak()
	if currentStreak >= 3 {
		score += 20
	}

	var weeklyTasks int
	_ = database.DB.QueryRow("SELECT COUNT(*) FROM goals WHERE claimed = 1 AND date(claimed_at) >= date('now', '-7 days')").Scan(&weeklyTasks)

	if weeklyTasks >= 50 {
		score += 30
	} else if weeklyTasks >= 25 {
		score += 15
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
