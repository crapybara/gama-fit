package main

import (
	"fmt"
	"html/template"
)

func main() {
	_, err := template.ParseFiles("templates/analytics.html")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("No syntax error in template")
	}
}
