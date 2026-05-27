package handlers

import (
	"io"
	"net/http"
	"os"

	"gama-fit/database"
)

func HandleExportDB(w http.ResponseWriter, r *http.Request) {
	// Flush WAL and close DB connection to ensure file is consistent
	if database.DB != nil {
		_, _ = database.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		_ = database.DB.Close()
	}

	dbPath := "./gamafit.db"
	file, err := os.Open(dbPath)
	if err != nil {
		http.Error(w, "Could not open database file", http.StatusInternalServerError)
		// Reconnect if we failed
		database.ConnectAndSetup()
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=gamafit.db")
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, file)

	// Reconnect after export
	database.ConnectAndSetup()
}

func HandleImportDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("database")
	if err != nil {
		http.Error(w, "Could not get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Flush and close DB
	if database.DB != nil {
		_, _ = database.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		_ = database.DB.Close()
	}

	dbPath := "./gamafit.db"
	// Backup current DB just in case
	_ = os.Rename(dbPath, dbPath+".bak")

	dst, err := os.Create(dbPath)
	if err != nil {
		http.Error(w, "Could not create database file", http.StatusInternalServerError)
		_ = os.Rename(dbPath+".bak", dbPath)
		database.ConnectAndSetup()
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Could not save database file", http.StatusInternalServerError)
		_ = os.Rename(dbPath+".bak", dbPath)
		database.ConnectAndSetup()
		return
	}

	// Reconnect to the new DB
	database.ConnectAndSetup()

	// Redirect back to analytics with a success message (or just reload)
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database imported successfully!"))
}
