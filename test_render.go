package main

import (
	"html/template"
	"log"
	"os"
	"gama-fit/internal/analytics"
)

func main() {
	tmpl, err := template.ParseFiles("internal/templates/analytics.html")
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(os.Stdout, analytics.AnalyticsData{})
	if err != nil {
		log.Fatal(err)
	}
}
