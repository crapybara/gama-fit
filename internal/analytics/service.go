package analytics

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gama-fit/database"
)

type ChartPoint struct {
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
	Reps   int     `json:"reps,omitempty"`
}

func FetchUserExercises(userID int) []string {
	var exercises []string
	rows, err := database.DB.Query(`SELECT DISTINCT exercise_name FROM freestyle_logs WHERE user_id = $1 ORDER BY exercise_name`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			exercises = append(exercises, name)
		}
	}
	return exercises
}

func FetchYearExercisePoints(userID int, exercise string, selectedYear int) []ChartPoint {
	var points []ChartPoint
	if exercise == "" {
		return points
	}
	query := `
		SELECT logged_date, AVG(weight)
		FROM freestyle_logs 
		WHERE user_id = $1 AND exercise_name = $2 AND logged_date LIKE $3
		GROUP BY logged_date
		ORDER BY logged_date ASC
	`
	rows, err := database.DB.Query(query, userID, exercise, fmt.Sprintf("%d-%%", selectedYear))
	if err != nil {
		return points
	}
	defer rows.Close()

	for rows.Next() {
		var d string
		var w float64
		if err := rows.Scan(&d, &w); err == nil {
			t, _ := time.Parse("2006-01-02", d)
			points = append(points, ChartPoint{
				Label:  strings.ToLower(t.Format("02 Jan")),
				Weight: w,
			})
		}
	}
	return points
}

func FetchYearBodyWeightPoints(userID int, selectedYear int, firstLogDate time.Time) []ChartPoint {
	var points []ChartPoint
	query := `
		SELECT log_date, AVG(weight)
		FROM body_weight_logs 
		WHERE user_id = $1 AND log_date LIKE $2
		GROUP BY log_date
		ORDER BY log_date ASC
	`
	rows, err := database.DB.Query(query, userID, fmt.Sprintf("%d-%%", selectedYear))
	if err != nil {
		return points
	}
	defer rows.Close()

	for rows.Next() {
		var d string
		var w float64
		if err := rows.Scan(&d, &w); err == nil {
			t, _ := time.Parse("2006-01-02", d)
			if t.Before(firstLogDate) && t.Format("2006-01-02") != firstLogDate.Format("2006-01-02") {
				continue
			}
			points = append(points, ChartPoint{
				Label:  strings.ToLower(t.Format("02 Jan")),
				Weight: math.Round(w*10)/10,
			})
		}
	}
	return points
}

func FetchFirstLogDate(userID int) time.Time {
	var dates []string
	queries := []string{
		"SELECT MIN(logged_date) FROM freestyle_logs WHERE user_id = $1",
		"SELECT MIN(log_date) FROM body_weight_logs WHERE user_id = $1",
		"SELECT MIN(log_date) FROM daily_meals WHERE user_id = $1",
		"SELECT MIN(log_date) FROM sleep_logs WHERE user_id = $1",
	}

	for _, q := range queries {
		var d sql.NullString
		_ = database.DB.QueryRow(q, userID).Scan(&d)
		if d.Valid && d.String != "" {
			dates = append(dates, d.String)
		}
	}

	if len(dates) == 0 {
		return time.Now()
	}

	sort.Strings(dates)
	t, _ := time.Parse("2006-01-02", dates[0])
	return t
}

func FetchAverageSleepHours(userID int) float64 {
	var avg float64
	_ = database.DB.QueryRow(`SELECT COALESCE(AVG(duration_mins), 0) / 60.0 FROM sleep_logs WHERE user_id = $1 AND log_date >= TO_CHAR(CURRENT_DATE - INTERVAL '6 days', 'YYYY-MM-DD')`, userID).Scan(&avg)
	return avg
}

func FetchAverageNutrition(userID int) (int, int) {
	var totalCalories int
	var totalProtein float64
	err := database.DB.QueryRow(`
		SELECT COALESCE(SUM(calories), 0), COALESCE(SUM(protein), 0.0) 
		FROM daily_meals 
		WHERE user_id = $1 AND log_date >= TO_CHAR(CURRENT_DATE - INTERVAL '6 days', 'YYYY-MM-DD')
	`, userID).Scan(&totalCalories, &totalProtein)
	if err != nil {
		return 0, 0
	}
	return totalCalories / 7, int(math.Round(totalProtein / 7.0))
}