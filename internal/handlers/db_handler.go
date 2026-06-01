package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func HandleExportDB(w http.ResponseWriter, r *http.Request) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5433/gamafit?sslmode=disable"
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("pg_dump", dsn)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Export failed.\n\n")
		fmt.Fprintf(w, "If you are using Docker, you can also run this command in your terminal:\n")
		fmt.Fprintf(w, "docker exec gama-fit-db pg_dump -U user gamafit > backup.sql\n\n")
		fmt.Fprintf(w, "Details: %v\n%s", err, stderr.String())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gamafit_backup_%s.sql", time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "application/sql")
	w.Write(stdout.Bytes())
}

func HandleImportDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("database")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "gamafit_import_*.sql")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.ReadFrom(file); err != nil {
		http.Error(w, "Failed to save temp file", http.StatusInternalServerError)
		return
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://user:password@localhost:5433/gamafit?sslmode=disable"
	}

	var stderr bytes.Buffer
	cmd := exec.Command("psql", dsn, "-f", tempFile.Name())
	cmd.Stderr = &stderr
	err = cmd.Run()

	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Import failed.\n\n")
		fmt.Fprintf(w, "If you are using Docker, you can also run this manually:\n")
		fmt.Fprintf(w, "docker cp <your_file.sql> gama-fit-db:/tmp/restore.sql\n")
		fmt.Fprintf(w, "docker exec gama-fit-db psql -U user -d gamafit -f /tmp/restore.sql\n\n")
		fmt.Fprintf(w, "Details: %v\n%s", err, stderr.String())
		return
	}

	http.Redirect(w, r, "/settings.html", http.StatusSeeOther)
}
