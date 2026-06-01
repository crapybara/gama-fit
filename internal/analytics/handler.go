package analytics

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gama-fit/handlers"
)

type AnalyticsData struct {
	SelectedYear         int
	SelectedRange        string
	SelectedExercise     string
	Exercises            []string
	Years                []int
	AvgSleep             string
	AvgCalories          int
	AvgProtein           int
	ExercisePointsJSON   template.JS
	BodyWeightPointsJSON template.JS
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
		start = now.AddDate(0, 0, -7)
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

	avgSleepHours := FetchAverageSleepHours(userID, start, end)
	avgCalories, avgProtein := FetchAverageNutrition(userID, start, end)

	exPoints := FetchExercisePoints(userID, selectedExercise, start, end)
	bwPoints := FetchBodyWeightPoints(userID, start, end)

	exJSON, _ := json.Marshal(exPoints)
	bwJSON, _ := json.Marshal(bwPoints)

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
	}

	// Save to cache (expiring in 5 minutes)
	cacheMu.Lock()
	analyticsCache[cacheKey] = cacheEntry{data: data, exp: time.Now().Add(5 * time.Minute)}
	cacheMu.Unlock()

	renderAnalytics(w, data)
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