package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func GetDSN() string {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is required")
	}
	return dsn
}

func ConnectAndSetup() {
	var err error

	dsn := GetDSN()

	// Retry connection because the database might not be ready immediately in Docker
	for i := 0; i < 10; i++ {
		DB, err = sql.Open("postgres", dsn)
		if err == nil {
			err = DB.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Failed to connect to postgres (attempt %d): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database after retries:", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	setupQueries := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TIMESTAMP NOT NULL
	);

	-- Robust migration: Add user_id to existing tables if it's missing
	DO $$ 
	BEGIN 
		-- goals
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'goals') THEN
			CREATE TABLE goals (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				title TEXT NOT NULL,
				completed INTEGER NOT NULL DEFAULT 0,
				claimed INTEGER NOT NULL DEFAULT 0,
				reward INTEGER NOT NULL DEFAULT 50,
				claimed_at TEXT
			);
		ELSE
			ALTER TABLE goals ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- user_stats
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'user_stats') THEN
			CREATE TABLE user_stats (
				user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
				total_coins INTEGER NOT NULL DEFAULT 0,
				current_streak INTEGER NOT NULL DEFAULT 0,
				bmi REAL DEFAULT 0,
				height REAL DEFAULT 0,
				neck REAL DEFAULT 0,
				belly REAL DEFAULT 0,
				arms REAL DEFAULT 0,
				calf REAL DEFAULT 0,
				age INTEGER DEFAULT 25,
				gender TEXT DEFAULT 'male',
				theme TEXT DEFAULT 'default',
				goal_weight REAL DEFAULT 0
			);
		ELSE
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS height REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS bmi REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS neck REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS belly REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS arms REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS calf REAL DEFAULT 0;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS age INTEGER DEFAULT 25;
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS gender TEXT DEFAULT 'male';
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS theme TEXT DEFAULT 'default';
			ALTER TABLE user_stats ADD COLUMN IF NOT EXISTS goal_weight REAL DEFAULT 0;
		END IF;

		-- cardio_logs
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'cardio_logs') THEN
			CREATE TABLE cardio_logs (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				heart_rate INTEGER NOT NULL,
				duration INTEGER NOT NULL, -- in minutes
				pace TEXT NOT NULL,
				intensity TEXT,
				logged_date TEXT NOT NULL DEFAULT CURRENT_DATE::TEXT,
				logged_time TEXT NOT NULL DEFAULT (TO_CHAR(CURRENT_TIMESTAMP, 'HH24:MI:SS'))
			);
		ELSE
			ALTER TABLE cardio_logs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- gym_logs
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'gym_logs') THEN
			CREATE TABLE gym_logs (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				log_date TEXT NOT NULL,
				content TEXT NOT NULL,
				UNIQUE(user_id, log_date)
			);
		ELSE
			ALTER TABLE gym_logs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- checkins
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'checkins') THEN
			CREATE TABLE checkins (
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				checkin_date TEXT NOT NULL,
				PRIMARY KEY (user_id, checkin_date)
			);
		ELSE
			ALTER TABLE checkins ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- freestyle_logs
		if NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'freestyle_logs') THEN
			CREATE TABLE freestyle_logs (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				exercise_name TEXT NOT NULL,
				weight REAL NOT NULL,
				reps INTEGER NOT NULL,
				sets INTEGER DEFAULT 1,
				muscle TEXT,
				is_cardio INTEGER DEFAULT 0,
				logged_date TEXT NOT NULL DEFAULT CURRENT_DATE::TEXT,
				logged_time TEXT NOT NULL DEFAULT (TO_CHAR(CURRENT_TIMESTAMP, 'HH24:MI:SS'))
			);
		ELSE
			ALTER TABLE freestyle_logs ADD COLUMN IF NOT EXISTS muscle TEXT;
			ALTER TABLE freestyle_logs ADD COLUMN IF NOT EXISTS is_cardio INTEGER DEFAULT 0;
			ALTER TABLE freestyle_logs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
			ALTER TABLE freestyle_logs ADD COLUMN IF NOT EXISTS sets INTEGER DEFAULT 1;
		END IF;


		-- workout_plans
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'workout_plans') THEN
			CREATE TABLE workout_plans (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				day_of_week INTEGER NOT NULL,
				exercise_name TEXT NOT NULL,
				sets INTEGER NOT NULL DEFAULT 3,
				reps TEXT NOT NULL DEFAULT '8-10',
				muscle TEXT,
				exercise_type TEXT
			);
		ELSE
			ALTER TABLE workout_plans ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
			ALTER TABLE workout_plans ADD COLUMN IF NOT EXISTS muscle TEXT;
			ALTER TABLE workout_plans ADD COLUMN IF NOT EXISTS exercise_type TEXT;
		END IF;

		-- user_macros_final
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'user_macros_final') THEN
			CREATE TABLE user_macros_final (
				user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
				calories INTEGER NOT NULL,
				protein REAL NOT NULL,
				carbs REAL NOT NULL,
				fats REAL NOT NULL
			);
		ELSE
			ALTER TABLE user_macros_final ALTER COLUMN protein TYPE REAL;
			ALTER TABLE user_macros_final ALTER COLUMN carbs TYPE REAL;
			ALTER TABLE user_macros_final ALTER COLUMN fats TYPE REAL;
		END IF;

		-- food_catalog
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'food_catalog') THEN
			CREATE TABLE food_catalog (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				name TEXT NOT NULL,
				calories INTEGER NOT NULL,
				protein REAL NOT NULL,
				carbs REAL NOT NULL,
				fats REAL NOT NULL,
				weight REAL NOT NULL DEFAULT 100,
				UNIQUE(user_id, name)
			);
		ELSE
			ALTER TABLE food_catalog ALTER COLUMN protein TYPE REAL;
			ALTER TABLE food_catalog ALTER COLUMN carbs TYPE REAL;
			ALTER TABLE food_catalog ALTER COLUMN fats TYPE REAL;
			ALTER TABLE food_catalog ALTER COLUMN weight TYPE REAL;
		END IF;

		-- daily_meals
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'daily_meals') THEN
			CREATE TABLE daily_meals (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				name TEXT NOT NULL,
				calories INTEGER NOT NULL,
				protein REAL NOT NULL,
				carbs REAL NOT NULL,
				fats REAL NOT NULL,
				log_date TEXT NOT NULL DEFAULT (CURRENT_DATE::TEXT),
				log_time TEXT NOT NULL DEFAULT (CURRENT_TIME::TEXT)
			);
		ELSE
			ALTER TABLE daily_meals ALTER COLUMN protein TYPE REAL;
			ALTER TABLE daily_meals ALTER COLUMN carbs TYPE REAL;
			ALTER TABLE daily_meals ALTER COLUMN fats TYPE REAL;
		END IF;

		-- body_weight_logs
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'body_weight_logs') THEN
			CREATE TABLE body_weight_logs (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				weight REAL NOT NULL,
				log_date TEXT NOT NULL,
				UNIQUE(user_id, log_date)
			);
		ELSE
			ALTER TABLE body_weight_logs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- sleep_logs
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'sleep_logs') THEN
			CREATE TABLE sleep_logs (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				log_date TEXT NOT NULL DEFAULT (CURRENT_DATE::TEXT),
				bedtime TEXT NOT NULL,
				waketime TEXT NOT NULL,
				quality TEXT NOT NULL,
				duration_mins INTEGER NOT NULL,
				score INTEGER NOT NULL,
				UNIQUE(user_id, log_date)
			);
		ELSE
			ALTER TABLE sleep_logs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;
		
		-- shop_catalog
		IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'shop_catalog') THEN
			CREATE TABLE shop_catalog (
				id SERIAL PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				category TEXT NOT NULL CHECK(category IN ('activity', 'item')),
				name TEXT NOT NULL,
				cost INTEGER NOT NULL,
				owned INTEGER NOT NULL DEFAULT 0,
				UNIQUE(user_id, name)
			);
		ELSE
			ALTER TABLE shop_catalog ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
		END IF;

		-- Add performance indexes
		CREATE INDEX IF NOT EXISTS idx_freestyle_logs_user_exercise_date ON freestyle_logs(user_id, exercise_name, logged_date);
		CREATE INDEX IF NOT EXISTS idx_body_weight_logs_user_date ON body_weight_logs(user_id, log_date);
		CREATE INDEX IF NOT EXISTS idx_daily_meals_user_date ON daily_meals(user_id, log_date);
		CREATE INDEX IF NOT EXISTS idx_sleep_logs_user_date ON sleep_logs(user_id, log_date);
	END $$;
	`

	_, err = DB.Exec(setupQueries)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	log.Println("PostgreSQL database connected and ready!")
}
