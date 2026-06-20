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

	// 1. Sleep (20 pts)
	sleepScore := float64(calculate14DaySleepScore(userID))
	ptsSleep := (sleepScore / 100.0) * 20.0

	// 2. Body Comp (20 pts)
	ptsBody := 0.0
	if weight > 0 && height > 0 {
		hMeter := height / 100.0
		bmi := weight / (hMeter * hMeter)
		diff := math.Abs(bmi - 22.0)
		ptsBody = 20.0 * math.Exp(-0.5*math.Pow(diff/4.0, 2))
	} else {
		ptsBody = 15.0 // fallback
	}

	// 3. Consistency (20 pts)
	ptsStreak := math.Min(20.0, (float64(currentStreak)/14.0)*20.0)

	// 4. Nutrition (Calories & Protein) (20 pts)
	var targetCals int
	var targetPro float64
	_ = database.DB.QueryRow("SELECT calories, protein FROM user_macros_final WHERE user_id = $1 ORDER BY id DESC LIMIT 1", userID).Scan(&targetCals, &targetPro)

	var avgCals int
	var avgPro float64
	_ = database.DB.QueryRow(`
		SELECT COALESCE(AVG(calories), 0), COALESCE(AVG(protein), 0)
		FROM daily_meals
		WHERE user_id = $1 AND log_date >= TO_CHAR($2::DATE - INTERVAL '7 days', 'YYYY-MM-DD')
	`, userID, localDate).Scan(&avgCals, &avgPro)
	
	ptsNutrition := 0.0
	if targetCals > 0 && targetPro > 0 {
		calPct := float64(avgCals) / float64(targetCals)
		proPct := float64(avgPro) / targetPro
		calScore := 10.0 * math.Exp(-0.5*math.Pow((calPct-1.0)/0.2, 2))
		proScore := 10.0
		if proPct < 1.0 { proScore = proPct * 10.0 }
		ptsNutrition = calScore + proScore
	} else {
		ptsNutrition = 15.0
	}

	// 5. Activity & Volume Goal (20 pts)
	var plannedVol int
	_ = database.DB.QueryRow("SELECT COALESCE(SUM(sets * target_reps * weight), 0) FROM workout_plan WHERE user_id = $1", userID).Scan(&plannedVol)
	
	var loggedVol int
	_ = database.DB.QueryRow(`
		SELECT COALESCE(SUM(weight * reps), 0) FROM freestyle_logs
		WHERE user_id = $1 AND logged_date >= TO_CHAR($2::DATE - INTERVAL '7 days', 'YYYY-MM-DD')
	`, userID, localDate).Scan(&loggedVol)
	
	ptsVolume := 0.0
	if plannedVol > 0 {
		volPct := float64(loggedVol) / float64(plannedVol)
		if volPct > 1.0 { volPct = 1.0 }
		ptsVolume = volPct * 20.0
	} else {
		var activeDays int
		_ = database.DB.QueryRow("SELECT COUNT(DISTINCT logged_date) FROM freestyle_logs WHERE user_id = $1 AND logged_date >= TO_CHAR($2::DATE - INTERVAL '14 days', 'YYYY-MM-DD')", userID, localDate).Scan(&activeDays)
		ptsVolume = math.Min(20.0, (float64(activeDays)/8.0)*20.0)
	}

	total := int(math.Round(ptsSleep + ptsBody + ptsStreak + ptsNutrition + ptsVolume))
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
