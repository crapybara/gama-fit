package handlers

import (
	"fmt"
	"net/http"
	"time"

	"gama-fit/database"
)

func GetTrueStreak(userID int, todayStr string) int {
	rows, err := database.DB.Query("SELECT checkin_date FROM checkins WHERE user_id = $1", userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	allCheckins := make(map[string]bool)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			allCheckins[d] = true
		}
	}

	realStreak := 0
	now, _ := time.Parse("2006-01-02", todayStr)
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

	_, _ = database.DB.Exec("UPDATE user_stats SET current_streak = $1 WHERE user_id = $2", realStreak, userID)
	return realStreak
}

func GetStreak(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	todayStr, _ := getLocalTime(r)
	streak := GetTrueStreak(userID, todayStr)

	ringPercentage := (float64(streak) / 365.0) * 100
	if ringPercentage > 100 {
		ringPercentage = 100
	}

	html := fmt.Sprintf(`
		<span id="streak-days" class="font-display text-5xl font-black text-white">%d</span>
		<script>
			setTimeout(() => {
				if(typeof window.animateRing === 'function') {
					window.animateRing('streakRing', %f);
				}
			}, 50);
		</script>
	`, streak, ringPercentage)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
