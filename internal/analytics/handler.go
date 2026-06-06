package analytics

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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
	ThisWeekVolume   float64
	VolumeChange     float64
	BestLift         BestLift
	
	// Body Composition
	BMI  float64
	FFMI float64
	LBM  float64

	// Focus Stats
	Focus          FocusSummary
	FocusChartJSON template.JS
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

	bmi, ffmi, lbm := FetchBodyComposition(userID)

	exPoints := FetchExercisePoints(userID, selectedExercise, start, end)
	bwPoints := FetchBodyWeightPoints(userID, start, end)

	// Calculate Focus week (Monday to Sunday) for current week
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}
	currentMonday := now.AddDate(0, 0, -daysSinceMonday)
	focusStart := currentMonday
	focusEnd := focusStart.AddDate(0, 0, 6)
	
	// Ensure bounds
	focusStart = time.Date(focusStart.Year(), focusStart.Month(), focusStart.Day(), 0, 0, 0, 0, time.Local)
	focusEnd = time.Date(focusEnd.Year(), focusEnd.Month(), focusEnd.Day(), 23, 59, 59, 0, time.Local)

	focusStats := GetFocusStats(userID, focusStart, focusEnd)

	exJSON, _ := json.Marshal(exPoints)
	bwJSON, _ := json.Marshal(bwPoints)
	focusJSON, _ := json.Marshal(focusStats.DailyChart)

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
		ThisWeekVolume:   thisWeekVol,
		VolumeChange:     volChange,
		BestLift:         bestLift,
		
		// Body Composition
		BMI:  bmi,
		FFMI: ffmi,
		LBM:  lbm,

		// Focus Stats
		Focus:          focusStats,
		FocusChartJSON: template.JS(focusJSON),
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
		err := handlers.Templates.ExecuteTemplate(w, "analytics.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Templates not initialized", http.StatusInternalServerError)
	}
}