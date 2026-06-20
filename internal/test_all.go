package main

import (
	"fmt"
	"html/template"
	"path/filepath"
)

func main() {
	files, _ := filepath.Glob("templates/*.html")
	_, err := template.ParseFiles(files...)
	if err != nil {
		fmt.Println("TEMPLATE ERROR:", err)
	} else {
		fmt.Println("NO TEMPLATE ERRORS")
	}
}
