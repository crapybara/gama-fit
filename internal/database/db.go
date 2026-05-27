package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func ConnectAndSetup() {
	var err error

	DB, err = sql.Open("sqlite", "./gamafit.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)

	setupQueries := `
	PRAGMA journal_mode=WAL;
	PRAGMA foreign_keys=ON;
	PRAGMA busy_timeout=5000;

	CREATE TABLE IF NOT EXISTS goals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		completed INTEGER NOT NULL DEFAULT 0,
		claimed INTEGER NOT NULL DEFAULT 0,
		reward INTEGER NOT NULL DEFAULT 50,
		claimed_at TEXT
	);

	CREATE TABLE IF NOT EXISTS user_stats (
		id INTEGER PRIMARY KEY,
		total_coins INTEGER NOT NULL DEFAULT 4250,
		current_streak INTEGER NOT NULL DEFAULT 0
	);

	INSERT OR IGNORE INTO user_stats (id, total_coins, current_streak)
	VALUES (1, 4250, 0);

	CREATE TABLE IF NOT EXISTS checkins (
		checkin_date TEXT PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS freestyle_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		exercise_name TEXT NOT NULL,
		weight REAL NOT NULL,
		reps INTEGER NOT NULL,
		logged_date TEXT NOT NULL DEFAULT (date('now')),
		logged_time TEXT NOT NULL DEFAULT (time('now'))
	);

	CREATE TABLE IF NOT EXISTS workout_plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		day_of_week INTEGER NOT NULL,
		exercise_name TEXT NOT NULL,
		sets INTEGER NOT NULL DEFAULT 3,
		reps TEXT NOT NULL DEFAULT '8-10'
	);

	CREATE TABLE IF NOT EXISTS creatine_tracker_final (
		log_date TEXT PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS user_macros_final (
		id INTEGER PRIMARY KEY,
		calories INTEGER NOT NULL,
		protein INTEGER NOT NULL,
		carbs INTEGER NOT NULL,
		fats INTEGER NOT NULL
	);

	INSERT OR IGNORE INTO user_macros_final (id, calories, protein, carbs, fats)
	VALUES (1, 2500, 200, 300, 70);

	CREATE TABLE IF NOT EXISTS food_catalog (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		calories INTEGER NOT NULL,
		protein INTEGER NOT NULL,
		carbs INTEGER NOT NULL,
		fats INTEGER NOT NULL,
		weight INTEGER NOT NULL DEFAULT 100
	);

	CREATE TABLE IF NOT EXISTS daily_meals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		calories INTEGER NOT NULL,
		protein INTEGER NOT NULL,
		carbs INTEGER NOT NULL,
		fats INTEGER NOT NULL,
		log_date TEXT NOT NULL DEFAULT (date('now')),
		log_time TEXT NOT NULL DEFAULT (time('now'))
	);

	CREATE TABLE IF NOT EXISTS sleep_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		log_date TEXT UNIQUE NOT NULL DEFAULT (date('now')),
		bedtime TEXT NOT NULL,
		waketime TEXT NOT NULL,
		quality TEXT NOT NULL,
		duration_mins INTEGER NOT NULL,
		score INTEGER NOT NULL
	);
	
	CREATE TABLE IF NOT EXISTS shop_catalog (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category TEXT NOT NULL CHECK(category IN ('activity', 'item')),
		name TEXT NOT NULL UNIQUE,
		cost INTEGER NOT NULL,
		owned INTEGER NOT NULL DEFAULT 0
	);
	`

	_, err = DB.Exec(setupQueries)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	log.Println("SQLite database connected and ready!")
}
