package analytics

import (
	"database/sql"
	"math"
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

func FetchExercisePoints(userID int, exercise string, start, end time.Time) []ChartPoint {
	var points []ChartPoint
	if exercise == "" {
		return points
	}
	query := `
		SELECT logged_date, AVG(weight)
		FROM freestyle_logs 
		WHERE user_id = $1 AND exercise_name = $2 AND logged_date >= $3 AND logged_date <= $4
		GROUP BY logged_date
		ORDER BY logged_date ASC
	`
	rows, err := database.DB.Query(query, userID, exercise, start.Format("2006-01-02"), end.Format("2006-01-02"))
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

func FetchBodyWeightPoints(userID int, start, end time.Time) []ChartPoint {
	var points []ChartPoint
	query := `
		SELECT log_date, AVG(weight)
		FROM body_weight_logs 
		WHERE user_id = $1 AND log_date >= $2 AND log_date <= $3
		GROUP BY log_date
		ORDER BY log_date ASC
	`
	rows, err := database.DB.Query(query, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
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
				Weight: math.Round(w*10)/10,
			})
		}
	}
	return points
}

func FetchFirstLogDate(userID int) time.Time {
	query := `
		SELECT MIN(d) FROM (
			SELECT MIN(logged_date) as d FROM freestyle_logs WHERE user_id = $1
			UNION ALL
			SELECT MIN(log_date) FROM body_weight_logs WHERE user_id = $1
			UNION ALL
			SELECT MIN(log_date) FROM daily_meals WHERE user_id = $1
			UNION ALL
			SELECT MIN(log_date) FROM sleep_logs WHERE user_id = $1
		) AS combined_dates
	`
	var d sql.NullString
	_ = database.DB.QueryRow(query, userID).Scan(&d)

	if !d.Valid || d.String == "" {
		return time.Now()
	}

	t, _ := time.Parse("2006-01-02", d.String)
	return t
}

func FetchAverageSleepHours(userID int, start, end time.Time) float64 {
	var avg float64
	query := `SELECT COALESCE(AVG(duration_mins), 0) / 60.0 FROM sleep_logs WHERE user_id = $1 AND log_date >= $2 AND log_date <= $3`
	_ = database.DB.QueryRow(query, userID, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&avg)
	return avg
}

func FetchAverageNutrition(userID int, start, end time.Time) (int, int) {
	var totalCalories int
	var totalProtein float64
	query := `
		SELECT COALESCE(SUM(calories), 0), COALESCE(SUM(protein), 0.0) 
		FROM daily_meals 
		WHERE user_id = $1 AND log_date >= $2 AND log_date <= $3
	`
	err := database.DB.QueryRow(query, userID, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&totalCalories, &totalProtein)
	if err != nil {
		return 0, 0
	}

	days := int(end.Sub(start).Hours()/24) + 1
	if days <= 0 {
		days = 1
	}
	return totalCalories / days, int(math.Round(totalProtein / float64(days)))
}
