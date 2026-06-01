package analytics

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"gama-fit/handlers"
)

type AnalyticsData struct {
	SelectedYear         int
	SelectedExercise     string
	Exercises            []string
	Years                []int
	AvgSleep             string
	AvgCalories          int
	AvgProtein           int
	ExercisePointsJSON   template.JS
	BodyWeightPointsJSON template.JS
}

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

	selectedExercise := r.URL.Query().Get("exercise")
	exercises := FetchUserExercises(userID)
	if selectedExercise == "" && len(exercises) > 0 {
		selectedExercise = exercises[0]
	}

	firstLogDate := FetchFirstLogDate(userID)
	startYear := firstLogDate.Year()
	endYear := time.Now().Year()
	if startYear == 0 || startYear > endYear {
		startYear = endYear
	}
	var years []int
	for y := startYear; y <= endYear; y++ {
		years = append(years, y)
	}

	avgSleepHours := FetchAverageSleepHours(userID)
	avgCalories, avgProtein := FetchAverageNutrition(userID)

	exPoints := FetchYearExercisePoints(userID, selectedExercise, selectedYear)
	bwPoints := FetchYearBodyWeightPoints(userID, selectedYear, firstLogDate)

	exJSON, _ := json.Marshal(exPoints)
	bwJSON, _ := json.Marshal(bwPoints)

	data := AnalyticsData{
		SelectedYear:         selectedYear,
		SelectedExercise:     selectedExercise,
		Exercises:            exercises,
		Years:                years,
		AvgSleep:             fmt.Sprintf("%.2f", avgSleepHours),
		AvgCalories:          avgCalories,
		AvgProtein:           avgProtein,
		ExercisePointsJSON:   template.JS(exJSON),
		BodyWeightPointsJSON: template.JS(bwJSON),
	}

	if handlers.Templates != nil {
		err := handlers.Templates.ExecuteTemplate(w, "analytics.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "Templates not initialized", http.StatusInternalServerError)
	}
}