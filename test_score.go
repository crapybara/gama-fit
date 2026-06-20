package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"gama-fit/internal/handlers"
	"gama-fit/internal/database"
)

func main() {
	database.InitDB()
	req, _ := http.NewRequest("GET", "/api/score?local_date=2026-06-20", nil)
	rr := httptest.NewRecorder()
	// Mock auth by passing user_id
	handlers.GetFitnessScore(rr, req)
	fmt.Println(rr.Body.String())
}
