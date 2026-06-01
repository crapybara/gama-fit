package database

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

func SeedDemoData() {
	if os.Getenv("SEED_DEMO_DATA") != "true" {
		log.Println("demo seed skipped: SEED_DEMO_DATA not true")
		return
	}

	if DB == nil {
		log.Println("demo seed skipped: database not ready")
		return
	}

	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	// Seed the panels that must never be empty first.
	seedVisibleDemoData(rng, now)

	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM checkins").Scan(&count); err == nil && count > 0 {
		log.Println("demo data already exists; skipped duplicate core seed")
		return
	}

	seedUserStats()
	seedMacros()
	seedGoals(now)
	seedCheckins(now)
	seedWorkoutPlans()

	log.Println("demo data seeded")
}

func seedVisibleDemoData(rng *rand.Rand, now time.Time) {
	var count int

	if err := DB.QueryRow("SELECT COUNT(*) FROM shop_catalog").Scan(&count); err == nil && count == 0 {
		seedShopCatalog()
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM food_catalog").Scan(&count); err == nil && count == 0 {
		seedFoodCatalog()
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM daily_meals").Scan(&count); err == nil && count == 0 {
		seedMeals(rng, now)
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM sleep_logs").Scan(&count); err == nil && count == 0 {
		seedSleepLogs(rng, now)
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM freestyle_logs").Scan(&count); err == nil && count < 220 {
		seedFreestyleLogs(rng, now)
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM body_weight_logs").Scan(&count); err == nil && count == 0 {
		seedBodyWeightLogs(rng, now)
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM user_stats").Scan(&count); err == nil && count == 0 {
		seedUserStats()
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM user_macros_final").Scan(&count); err == nil && count == 0 {
		seedMacros()
	}
}

func seedUserStats() {
	_, _ = DB.Exec(`INSERT INTO user_stats (user_id, total_coins, current_streak, height) VALUES (1, 0, 7, 175.0) ON CONFLICT (user_id) DO UPDATE SET total_coins = EXCLUDED.total_coins, current_streak = EXCLUDED.current_streak, height = EXCLUDED.height`)
}

func seedMacros() {
	_, _ = DB.Exec(`INSERT INTO user_macros_final (user_id, calories, protein, carbs, fats) VALUES (1, 2500, 200, 300, 70) ON CONFLICT (user_id) DO UPDATE SET calories = EXCLUDED.calories, protein = EXCLUDED.protein, carbs = EXCLUDED.carbs, fats = EXCLUDED.fats`)
}

func seedGoals(now time.Time) {
	type goal struct {
		Title     string
		Completed int
		Claimed   int
		Reward    int
	}
	goals := []goal{
		{"Hit 10k steps", 1, 1, 35},
		{"Drink 3L water", 1, 1, 25},
		{"Protein target", 1, 0, 50},
		{"Sleep before 11 PM", 0, 0, 40},
		{"Train upper body", 1, 1, 60},
		{"No junk food today", 0, 0, 45},
	}
	for i, g := range goals {
		claimedAt := ""
		if g.Claimed == 1 {
			claimedAt = now.AddDate(0, 0, -i).Format("2006-01-02")
		}
		_, err := DB.Exec(
			`INSERT INTO goals (title, completed, claimed, reward, claimed_at) VALUES ($1, $2, $3, $4, $5)`,
			g.Title, g.Completed, g.Claimed, g.Reward, claimedAt,
		)
		if err != nil {
			log.Printf("seed goal warning: %v", err)
		}
	}
}

func seedCheckins(now time.Time) {
	days := []int{0, 1, 2, 4, 5, 6, 8, 9, 10, 12, 13}
	for _, d := range days {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		_, err := DB.Exec(`INSERT INTO checkins (checkin_date) VALUES ($1) ON CONFLICT DO NOTHING`, date)
		if err != nil {
			log.Printf("seed checkin warning: %v", err)
		}
	}
}

func seedSleepLogs(rng *rand.Rand, now time.Time) {
	qualities := []string{"Good", "Average", "Great", "Poor", "Good", "Excellent", "Solid"}
	for i := 0; i < 14; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		hours := 6 + rng.Intn(3) // 6-8h
		minutes := hours*60 + rng.Intn(50)
		score := 58 + rng.Intn(35)

		bedHour := 22 + rng.Intn(3)
		if bedHour == 24 {
			bedHour = 0
		}
		bedtime := fmt.Sprintf("%02d:%02d:00", bedHour, rng.Intn(60))
		waketime := fmt.Sprintf("%02d:%02d:00", 6+rng.Intn(3), rng.Intn(60))
		quality := qualities[i%len(qualities)]

		_, err := DB.Exec(
			`INSERT INTO sleep_logs (log_date, bedtime, waketime, quality, duration_mins, score)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (log_date) DO UPDATE SET bedtime = EXCLUDED.bedtime, waketime = EXCLUDED.waketime, quality = EXCLUDED.quality, duration_mins = EXCLUDED.duration_mins, score = EXCLUDED.score`,
			date, bedtime, waketime, quality, minutes, score,
		)
		if err != nil {
			log.Printf("seed sleep warning: %v", err)
		}
	}
}

func seedFoodCatalog() {
	items := []struct {
		Name     string
		Calories int
		Protein  int
		Carbs    int
		Fats     int
	}{
		{"Oats Bowl", 320, 12, 48, 8},
		{"Chicken Rice Bowl", 540, 42, 58, 16},
		{"Greek Yogurt", 150, 15, 10, 4},
		{"Banana", 105, 1, 27, 0},
		{"Egg Omelette", 210, 18, 2, 15},
		{"Paneer Wrap", 430, 24, 35, 20},
		{"Peanut Butter Toast", 280, 10, 26, 14},
		{"Rice + Dal", 390, 16, 62, 7},
	}
	for _, item := range items {
		_, err := DB.Exec(
			`INSERT INTO food_catalog (name, calories, protein, carbs, fats) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
			item.Name, item.Calories, item.Protein, item.Carbs, item.Fats,
		)
		if err != nil {
			log.Printf("seed food catalog warning: %v", err)
		}
	}
}

func seedMeals(rng *rand.Rand, now time.Time) {
	meals := []string{
		"Oats Bowl", "Chicken Rice Bowl", "Greek Yogurt", "Banana",
		"Egg Omelette", "Paneer Wrap", "Peanut Butter Toast", "Rice + Dal",
	}
	for i := 0; i < 14; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		for j := 0; j < 2; j++ {
			var cal, pro, carb, fat int
			name := meals[rng.Intn(len(meals))]
			_ = DB.QueryRow(`SELECT calories, protein, carbs, fats FROM food_catalog WHERE name = $1`, name).Scan(&cal, &pro, &carb, &fat)
			logTime := fmt.Sprintf("%02d:%02d:00", 8+j*5+rng.Intn(2), rng.Intn(60))
			_, err := DB.Exec(
				`INSERT INTO daily_meals (name, calories, protein, carbs, fats, log_date, log_time) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				name, cal, pro, carb, fat, date, logTime,
			)
			if err != nil {
				log.Printf("seed meals warning: %v", err)
			}
		}
	}
}

func seedWorkoutPlans() {
	plans := []struct {
		Day  int
		Name string
		Sets int
		Reps string
	}{
		{1, "Bench Press", 4, "6-8"},
		{1, "Incline Dumbbell Press", 3, "8-10"},
		{2, "Squat", 4, "5-8"},
		{2, "Leg Press", 3, "10-12"},
		{3, "Lat Pulldown", 4, "8-12"},
		{3, "Seated Row", 3, "10-12"},
		{4, "Overhead Press", 4, "6-8"},
		{5, "Romanian Deadlift", 4, "6-8"},
		{6, "Biceps Curl", 3, "10-15"},
		{6, "Triceps Pushdown", 3, "10-15"},
	}
	for _, p := range plans {
		_, err := DB.Exec(
			`INSERT INTO workout_plans (day_of_week, exercise_name, sets, reps) VALUES ($1, $2, $3, $4)`,
			p.Day, p.Name, p.Sets, p.Reps,
		)
		if err != nil {
			log.Printf("seed workout warning: %v", err)
		}
	}
}

func seedFreestyleLogs(rng *rand.Rand, now time.Time) {
	type lift struct {
		Name  string
		Base  float64
		Gain  float64
		Shift float64
	}

	lifts := []lift{
		{"Bench Press", 35, 22, 0.22},
		{"Squat", 50, 30, 0.28},
		{"Deadlift", 55, 35, 0.30},
		{"Overhead Press", 20, 12, 0.16},
		{"Barbell Row", 30, 18, 0.20},
		{"Incline Press", 22, 16, 0.18},
		{"Lat Pulldown", 35, 20, 0.22},
		{"Cable Row", 28, 16, 0.18},
		{"Dumbbell Curl", 7, 6, 0.10},
		{"Leg Press", 70, 45, 0.35},
	}

	insert := func(date time.Time, x lift, intensity float64) {
		weight := x.Base + (intensity * x.Gain) + ((rng.Float64() - 0.5) * 4)
		if weight < x.Base*0.7 {
			weight = x.Base * 0.7
		}
		weight = math.Round(weight*10) / 10

		reps := 5 + rng.Intn(10)
		logTime := fmt.Sprintf("%02d:%02d:00", 16+rng.Intn(5), rng.Intn(60))

		_, err := DB.Exec(
			`INSERT INTO freestyle_logs (exercise_name, weight, reps, logged_date, logged_time) VALUES ($1, $2, $3, $4, $5)`,
			x.Name, weight, reps, date.Format("2006-01-02"), logTime,
		)
		if err != nil {
			log.Printf("seed freestyle warning: %v", err)
		}
	}

	// 36 months of history for month/year/all-time charts.
	for monthsBack := 35; monthsBack >= 0; monthsBack-- {
		monthStart := now.AddDate(0, -monthsBack, 0)
		progress := float64(35-monthsBack) / 35.0

		for session := 0; session < 4; session++ {
			day := 1 + rng.Intn(24)
			date := time.Date(monthStart.Year(), monthStart.Month(), day, 18+rng.Intn(3), rng.Intn(60), 0, 0, now.Location())
			x := lifts[(monthsBack+session)%len(lifts)]
			intensity := x.Shift + progress
			insert(date, x, intensity)
		}
	}

	// Recent 90 days for the 7-day view.
	for daysBack := 0; daysBack < 90; daysBack++ {
		if rng.Float64() > 0.45 {
			continue
		}
		sessions := 1 + rng.Intn(2)
		progress := 1.0 - (float64(daysBack) / 90.0)
		for session := 0; session < sessions; session++ {
			date := now.AddDate(0, 0, -daysBack)
			x := lifts[rng.Intn(len(lifts))]
			intensity := x.Shift + progress
			insert(date, x, intensity)
		}
	}
}

func seedBodyWeightLogs(rng *rand.Rand, now time.Time) {
	for i := 0; i < 365; i++ {
		if rng.Float64() < 0.6 { // Don't log every single day
			continue
		}
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		// Simulate a slow weight loss journey over a year
		progress := float64(i) / 365.0
		weight := 85.0 - (progress * 10) + (rng.Float64()-0.5)*1.5
		_, err := DB.Exec(`INSERT INTO body_weight_logs (log_date, weight) VALUES ($1, $2) ON CONFLICT DO NOTHING`, date, math.Round(weight*10)/10)
		if err != nil {
			log.Printf("seed body weight warning: %v", err)
		}
	}
}

func seedShopCatalog() {
	items := []struct {
		Category string
		Name     string
		Cost     int
	}{
		{"activity", "Morning Walk", 25},
		{"activity", "Deep Stretch", 20},
		{"activity", "Study Sprint", 30},
		{"activity", "Meditation", 15},
		{"item", "Protein Bar", 40},
		{"item", "Shaker Bottle", 75},
		{"item", "Resistance Band", 120},
		{"item", "Creatine Jar", 180},
	}
	for _, item := range items {
		_, err := DB.Exec(
			`INSERT INTO shop_catalog (category, name, cost, owned) VALUES ($1, $2, $3, 0) ON CONFLICT DO NOTHING`,
			item.Category, item.Name, item.Cost,
		)
		if err != nil {
			log.Printf("seed shop warning: %v", err)
		}
	}
}
