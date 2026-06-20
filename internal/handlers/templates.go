package handlers

import (
	"html/template"
	"log"
)

var Templates *template.Template

func InitTemplates() {
	var err error
	Templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		Templates, err = template.ParseGlob("internal/templates/*.html")
		if err != nil {
			Templates, err = template.ParseGlob("../internal/templates/*.html")
			if err != nil {
				log.Printf("Warning: failed to load templates: %v", err)
			}
		}
	}
}
