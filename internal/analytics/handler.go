package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gama-fit/database"
	"gama-fit/handlers"
)

type AnalyticsData struct {
	SelectedYear         int
	SelectedRange        string
	SelectedExercise     string
	SelectedDate         string
	Exercises            []string
	Years                []int
	AvgSleep             string
	AvgCalories          int
	AvgProtein           int
	ExercisePointsJSON   template.JS
	BodyWeightPointsJSON template.JS

	// New Lifting Stats
	ThisWeekVolume float64
	VolumeChange   float64
	BestLift       BestLift

	// Body Composition
	BMI      float64
	FFMI     float64
	LeanMass float64
	BodyFat  float64

	// Goal Progress
	WeightProgressPct float64
	GoalWeight        float64
	CurrentWeight     float64
	WeeklyTargetCals  int
	WeeklyLoggedCals  int
	WeeklyTargetPro   int
	WeeklyLoggedPro   int
	PlannedVolume     int
	LoggedVolume      int
	PlannedSets       int
	LoggedSets        int
	PlannedExercises  int
	LoggedExercises   int

	// Percentages
	CalProgressPct   float64
	ProProgressPct   float64
	VolProgressPct   float64
	CalProgressWidth float64
	ProProgressWidth float64
	VolProgressWidth float64
	ExtraVolume      int
}

type cacheEntry struct {
	data AnalyticsData
	exp  time.Time
}

var (
	analyticsCache = make(map[string]cacheEntry)
	cacheMu        sync.RWMutex
)

func HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, err := handlers.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = fmt.Sprintf("%d", time.Now().Year())
	}
	selectedYear, _ := strconv.Atoi(yearStr)

	rangeParam := r.URL.Query().Get("range") // "1w", "1m", "3m", "6m", "year"
	if rangeParam == "" {
		rangeParam = "1w"
	}

	selectedExercise := r.URL.Query().Get("exercise")
	exercises := FetchUserExercises(userID)
	if selectedExercise == "" && len(exercises) > 0 {
		selectedExercise = exercises[0]
	}

	// Check cache
	cacheKey := fmt.Sprintf("%d-%d-%s-%s", userID, selectedYear, rangeParam, selectedExercise)
	cacheMu.RLock()
	entry, found := analyticsCache[cacheKey]
	cacheMu.RUnlock()

	if found && time.Now().Before(entry.exp) {
		renderAnalytics(w, entry.data)
		return
	}

	// Calculate time range
	var start, end time.Time
	now := time.Now()
	end = now

	switch rangeParam {
	case "1w":
		start = now.AddDate(0, 0, -6)
	case "1m":
		start = now.AddDate(0, -1, 0)
	case "3m":
		start = now.AddDate(0, -3, 0)
	case "6m":
		start = now.AddDate(0, -6, 0)
	default:
		start = time.Date(selectedYear, 1, 1, 0, 0, 0, 0, time.Local)
		end = time.Date(selectedYear, 12, 31, 23, 59, 59, 0, time.Local)
	}

	firstLogDate := FetchFirstLogDate(userID)
	startYear := firstLogDate.Year()
	currYear := time.Now().Year()
	if startYear == 0 || startYear > currYear {
		startYear = currYear
	}
	var years []int
	for y := startYear; y <= currYear; y++ {
		years = append(years, y)
	}

	// The charts respect the selected range
	avgSleepHours := FetchAverageSleepHours(userID, start, end)
	avgCalories, avgProtein := FetchAverageNutrition(userID, start, end)

	// Top stats strictly use a 7-day rolling window
	fixedThisWeekStart := now.AddDate(0, 0, -6)
	fixedLastWeekStart := now.AddDate(0, 0, -13)
	fixedLastWeekEnd := now.AddDate(0, 0, -7)

	thisWeekVol := FetchTotalVolume(userID, fixedThisWeekStart, now)
	lastWeekVol := FetchTotalVolume(userID, fixedLastWeekStart, fixedLastWeekEnd)

	volChange := 0.0
	if lastWeekVol > 0 {
		volChange = ((thisWeekVol - lastWeekVol) / lastWeekVol) * 100
	}

	bestLift := FetchBestLift(userID, fixedThisWeekStart, now)

	bmi, ffmi, lbm, bodyFat := FetchBodyComposition(userID)

	exPoints := FetchExercisePoints(userID, selectedExercise, start, end)
	bwPoints := FetchBodyWeightPoints(userID, start, end)

	exJSON, _ := json.Marshal(exPoints)
	bwJSON, _ := json.Marshal(bwPoints)

	// --- Goal Progress Calculations ---
	var startWeight, goalWeight, currentWeight float64
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 ORDER BY log_date ASC LIMIT 1", userID).Scan(&startWeight)
	_ = database.DB.QueryRow("SELECT goal_weight FROM user_stats WHERE user_id = $1", userID).Scan(&goalWeight)
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 ORDER BY log_date DESC LIMIT 1", userID).Scan(&currentWeight)

	weightProgress := 0.0
	if startWeight > 0 && goalWeight > 0 && startWeight != goalWeight {
		weightProgress = (currentWeight - startWeight) / (goalWeight - startWeight) * 100
		if weightProgress < 0 {
			weightProgress = 0
		}
		if weightProgress > 100 {
			weightProgress = 100
		}
	}

	var targetCal, targetPro int
	_ = database.DB.QueryRow("SELECT calories, protein FROM user_macros_final WHERE user_id = $1", userID).Scan(&targetCal, &targetPro)
	weeklyTargetCals := targetCal * 7
	weeklyTargetPro := targetPro * 7

	var weeklyLoggedCals, weeklyLoggedPro int
	_ = database.DB.QueryRow("SELECT COALESCE(SUM(calories),0), COALESCE(SUM(protein),0) FROM daily_meals WHERE user_id = $1 AND log_date >= $2", userID, fixedThisWeekStart.Format("2006-01-02")).Scan(&weeklyLoggedCals, &weeklyLoggedPro)

	var plannedVolume, plannedSets, plannedExercises int
	_ = database.DB.QueryRow(`
		SELECT 
			COALESCE(SUM(sets * CAST(SPLIT_PART(reps, '-', 1) AS INTEGER)), 0) as pv, 
			COALESCE(SUM(sets), 0) as ps, 
			COUNT(*) as pe 
		FROM workout_plans WHERE user_id = $1
	`, userID).Scan(&plannedVolume, &plannedSets, &plannedExercises)

	var loggedVolume, loggedSets, loggedExercises int
	_ = database.DB.QueryRow(`
		SELECT 
			COALESCE(SUM(sets * reps), 0) as lv, 
			COALESCE(SUM(sets), 0) as ls, 
			COUNT(DISTINCT exercise_name) as le 
		FROM freestyle_logs 
		WHERE user_id = $1 AND logged_date >= $2 AND is_cardio = 0
	`, userID, fixedThisWeekStart.Format("2006-01-02")).Scan(&loggedVolume, &loggedSets, &loggedExercises)

	data := AnalyticsData{
		SelectedYear:         selectedYear,
		SelectedRange:        rangeParam,
		SelectedExercise:     selectedExercise,
		Exercises:            exercises,
		Years:                years,
		AvgSleep:             fmt.Sprintf("%.2f", avgSleepHours),
		AvgCalories:          avgCalories,
		AvgProtein:           avgProtein,
		ExercisePointsJSON:   template.JS(exJSON),
		BodyWeightPointsJSON: template.JS(bwJSON),

		// New Stats
		ThisWeekVolume: thisWeekVol,
		VolumeChange:   volChange,
		BestLift:       bestLift,

		// Body Composition
		BMI:      bmi,
		FFMI:     ffmi,
		LeanMass: lbm,
		BodyFat:  bodyFat,

		// Goal Progress
		WeightProgressPct: weightProgress,
		GoalWeight:        goalWeight,
		CurrentWeight:     currentWeight,
		WeeklyTargetCals:  weeklyTargetCals,
		WeeklyLoggedCals:  weeklyLoggedCals,
		WeeklyTargetPro:   weeklyTargetPro,
		WeeklyLoggedPro:   weeklyLoggedPro,
		PlannedVolume:     plannedVolume,
		LoggedVolume:      loggedVolume,
		PlannedSets:       plannedSets,
		LoggedSets:        loggedSets,
		PlannedExercises:  plannedExercises,
		LoggedExercises:   loggedExercises,

		CalProgressPct: 0.0,
		ProProgressPct: 0.0,
		VolProgressPct: 0.0,
	}

	if data.WeeklyTargetCals > 0 {
		data.CalProgressPct = float64(data.WeeklyLoggedCals) / float64(data.WeeklyTargetCals) * 100
	}
	if data.WeeklyTargetPro > 0 {
		data.ProProgressPct = float64(data.WeeklyLoggedPro) / float64(data.WeeklyTargetPro) * 100
	}
	if data.PlannedVolume > 0 {
		data.VolProgressPct = float64(data.LoggedVolume) / float64(data.PlannedVolume) * 100
	}

	data.CalProgressWidth = math.Min(100, data.CalProgressPct)
	data.ProProgressWidth = math.Min(100, data.ProProgressPct)
	data.VolProgressWidth = math.Min(100, data.VolProgressPct)
	if data.LoggedVolume > data.PlannedVolume {
		data.ExtraVolume = data.LoggedVolume - data.PlannedVolume
	}

	// Save to cache (expiring in 5 minutes)
	cacheMu.Lock()
	analyticsCache[cacheKey] = cacheEntry{data: data, exp: time.Now().Add(5 * time.Minute)}
	cacheMu.Unlock()

	renderAnalytics(w, data)
}

func HandleMuscle1RM(w http.ResponseWriter, r *http.Request) {
	userID, err := handlers.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	muscle := r.URL.Query().Get("muscle")
	if muscle == "" {
		w.Write([]byte(`<div class="text-zinc-500 text-[10px] uppercase font-black text-center py-4">Select a muscle to view details</div>`))
		return
	}

	lifts := FetchMuscleExercises1RM(userID, muscle)
	if len(lifts) == 0 {
		w.Write([]byte(fmt.Sprintf(`<div class="text-zinc-500 text-[10px] uppercase font-black text-center py-4">No data for %s</div>`, muscle)))
		return
	}

	html := fmt.Sprintf(`<div class="animate-fade-in">
		<h4 class="text-blue-400 text-[10px] font-black uppercase tracking-[0.2em] mb-4 border-b border-blue-500/20 pb-2">%s Progression</h4>
		<div class="space-y-3 max-h-72 overflow-y-auto pr-2 custom-scrollbar">`, strings.ToUpper(muscle))
	for _, bl := range lifts {
		html += fmt.Sprintf(`
			<div class="flex items-center justify-between group bg-white/5 hover:bg-white/10 p-3 rounded-xl transition-all border border-transparent hover:border-blue-500/20">
				<div>
					<p class="text-white text-xs font-black uppercase tracking-tight">%s</p>
					<p class="text-[10px] text-zinc-500 font-bold uppercase tracking-widest">%v KG × %d</p>
				</div>
				<div class="text-right">
					<p class="text-[10px] text-zinc-600 font-black uppercase tracking-widest mb-0.5">Est. 1RM</p>
					<p class="text-white text-sm font-black font-mono">%v<span class="text-[8px] text-blue-500 ml-1">KG</span></p>
				</div>
			</div>`, bl.Exercise, bl.Weight, bl.Reps, math.Round(bl.OneRM*10)/10)
	}
	html += `</div></div>`

	w.Write([]byte(html))
}

func HandleAnalyticsHeatmap(w http.ResponseWriter, r *http.Request) {
	userID, err := handlers.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "1w"
	}

	var start, end time.Time
	now := time.Now()
	end = now

	switch rangeParam {
	case "1w":
		start = now.AddDate(0, 0, -6)
	case "1m":
		start = now.AddDate(0, -1, 0)
	case "3m":
		start = now.AddDate(0, -3, 0)
	case "6m":
		start = now.AddDate(0, -6, 0)
	default:
		start = now.AddDate(0, 0, -7)
	}

	heatmap, total, maxVol := FetchAnalyticsHeatmap(userID, start, end)

	response := struct {
		Muscles   map[string]AnalyticsMuscleStats `json:"muscles"`
		Total     AnalyticsMuscleStats            `json:"total"`
		WeeklyMax float64                         `json:"weeklyMax"`
	}{
		Muscles:   heatmap,
		Total:     total,
		WeeklyMax: maxVol,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func renderAnalytics(w http.ResponseWriter, data AnalyticsData) {
	if handlers.Templates != nil {
		var buf bytes.Buffer
		err := handlers.Templates.ExecuteTemplate(&buf, "analytics.html", data)
		if err != nil {
			fmt.Printf("TEMPLATE ERROR: %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(buf.Bytes())
	} else {
		http.Error(w, "Templates not initialized", http.StatusInternalServerError)
	}
}
