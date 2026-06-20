package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gama-fit/database"
)

func HandleTheme(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)

	if r.Method == http.MethodPost {
		theme := r.FormValue("theme")
		_, err := database.DB.Exec("UPDATE user_stats SET theme = $1 WHERE user_id = $2", theme, userID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("HX-Trigger", "themeUpdated")
		w.WriteHeader(http.StatusOK)
		return
	}

	var theme string
	err := database.DB.QueryRow("SELECT theme FROM user_stats WHERE user_id = $1", userID).Scan(&theme)
	if err != nil {
		theme = "default"
	}
	w.Write([]byte(theme))
}

func HandleSettings(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)

	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if r.FormValue("action") == "macros" {
			pro, _ := strconv.ParseFloat(r.FormValue("protein"), 64)
			carb, _ := strconv.ParseFloat(r.FormValue("carbs"), 64)
			fat, _ := strconv.ParseFloat(r.FormValue("fats"), 64)
			cal := int(pro*4 + carb*4 + fat*9)
			_, _ = database.DB.Exec(`
				INSERT INTO user_macros_final (user_id, calories, protein, carbs, fats) 
				VALUES ($1, $2, $3, $4, $5) 
				ON CONFLICT(user_id) DO UPDATE SET 
				calories = excluded.calories, protein = excluded.protein, carbs = excluded.carbs, fats = excluded.fats
			`, userID, cal, pro, carb, fat)
			w.Header().Set("HX-Trigger", "macrosUpdated")
		} else {
			metric := r.FormValue("metric")
			valueStr := r.FormValue("value")

			if metric == "gender" {
				_, _ = database.DB.Exec("UPDATE user_stats SET gender = $1 WHERE user_id = $2", valueStr, userID)
			} else {
				val, err := strconv.ParseFloat(valueStr, 64)
				if err == nil && val >= 0 {
					if metric == "bmi" || metric == "height" || metric == "neck" || metric == "belly" || metric == "arms" || metric == "calf" || metric == "age" || metric == "goal_weight" {
						query := fmt.Sprintf("UPDATE user_stats SET %s = $1 WHERE user_id = $2", metric)
						_, _ = database.DB.Exec(query, val, userID)
					}
				}
			}
			w.Header().Set("HX-Trigger", "metricUpdated")
		}
	}

	var bmi, height, neck, belly, arms, calf, goalWeight float64
	var age int
	var gender, theme string
	err := database.DB.QueryRow("SELECT bmi, height, neck, belly, arms, calf, age, gender, theme, goal_weight FROM user_stats WHERE user_id = $1", userID).Scan(&bmi, &height, &neck, &belly, &arms, &calf, &age, &gender, &theme, &goalWeight)
	if err != nil {
		bmi, height, neck, belly, arms, calf, goalWeight = 0, 0, 0, 0, 0, 0, 0
		age = 25
		gender = "male"
		theme = "default"
	}
	if gender == "" {
		gender = "male"
	}

	var tPro, tCarb, tFat float64
	_ = database.DB.QueryRow("SELECT protein, carbs, fats FROM user_macros_final WHERE user_id = $1", userID).Scan(&tPro, &tCarb, &tFat)

	formatVal := func(v float64) string {
		if v > 0 {
			return fmt.Sprintf("%.1f", v)
		}
		return "--"
	}

	html := fmt.Sprintf(`
		<div class="space-y-6 animate-fade-in-up pb-12">
			<!-- Daily Macros Target -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden border-app-yellow/20">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-app-yellow font-black uppercase tracking-wider text-sm">Nutrition Targets</h3>
				</div>
				<form hx-post="/api/settings" hx-target="#settings-container" hx-swap="innerHTML" class="grid grid-cols-1 sm:grid-cols-4 gap-4">
					<input type="hidden" name="action" value="macros">
					<div>
						<label class="text-[10px] uppercase font-bold text-zinc-500 tracking-wider mb-1.5 block">Protein (g)</label>
						<input type="number" step="0.1" name="protein" value="%.1f" required class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-zinc-100 outline-none focus:border-app-pink font-mono">
					</div>
					<div>
						<label class="text-[10px] uppercase font-bold text-zinc-500 tracking-wider mb-1.5 block">Carbs (g)</label>
						<input type="number" step="0.1" name="carbs" value="%.1f" required class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-zinc-100 outline-none focus:border-blue-500 font-mono">
					</div>
					<div>
						<label class="text-[10px] uppercase font-bold text-zinc-500 tracking-wider mb-1.5 block">Fats (g)</label>
						<input type="number" step="0.1" name="fats" value="%.1f" required class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-zinc-100 outline-none focus:border-emerald-500 font-mono">
					</div>
					<div class="flex items-end">
						<button type="submit" class="w-full bg-white text-black font-black py-3 rounded-xl hover:bg-zinc-200 transition-all uppercase tracking-widest text-xs">Save</button>
					</div>
				</form>
			</div>

			<!-- Theme Selector -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">Appearance</h3>
				</div>
				<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
					<button onclick="setTheme('default')" class="p-3 rounded-xl border border-white/5 bg-app-surface hover:border-app-pink transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Default</button>
					<button onclick="setTheme('dracula')" class="p-3 rounded-xl border border-white/5 bg-[#282a36] hover:border-[#bd93f9] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Dracula</button>
					<button onclick="setTheme('synthwave')" class="p-3 rounded-xl border border-white/5 bg-[#05010a] hover:border-[#ff2fa8] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Synthwave 84</button>
					<button onclick="setTheme('material')" class="p-3 rounded-xl border border-white/5 bg-[#0b0911] hover:border-[#ff2fa8] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Material UI</button>
					<button onclick="setTheme('gruvbox')" class="p-3 rounded-xl border border-white/5 bg-[#282828] hover:border-[#fb4934] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Gruvbox</button>
					<button onclick="setTheme('catppuccin')" class="p-3 rounded-xl border border-white/5 bg-[#1e1e2e] hover:border-[#f5c2e7] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Catppuccin</button>
				</div>
			</div>



			<!-- Body Metrics Form -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">Log Body Metrics</h3>
				</div>
				<form hx-post="/api/settings" hx-target="#settings-container" hx-swap="innerHTML" class="space-y-4">
					<div>
						<label class="block text-[10px] font-bold uppercase tracking-wider text-zinc-500 mb-2">Metric Type</label>
						<select name="metric" class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 transition-colors">
							<option value="age">Age (years)</option>
							<option value="gender">Sex (Male/Female)</option>
							<option value="height">Height (cm)</option>
							<option value="neck">Neck (cm)</option>
							<option value="belly">Belly/Waist (cm)</option>
							<option value="arms">Arms (cm)</option>
							<option value="calf">Calf (cm)</option>
							<option value="goal_weight">Goal Weight (kg)</option>
						</select>
						</div>
						<div>
						<label class="block text-[10px] font-bold uppercase tracking-wider text-zinc-500 mb-2">Value / Option</label>
						<input type="text" name="value" placeholder="e.g. 25, male, 180" required class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 transition-colors font-mono">
						</div>
					<button type="submit" class="w-full bg-blue-500 text-white font-bold px-6 py-3 rounded-xl hover:bg-blue-400 transition-all shadow-[0_0_12px_rgba(59,130,246,0.25)] text-xs uppercase tracking-wider">Save Metric</button>
				</form>
			</div>

			<!-- Database Management -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden border-emerald-500/20">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-emerald-400 font-black uppercase tracking-wider text-sm">Database Management</h3>
				</div>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<h4 class="text-white font-bold text-xs uppercase mb-3">Export Data</h4>
						<p class="text-zinc-500 text-[10px] mb-4">Download a full backup of your database as a .sql file.</p>
						<a href="/api/db/export" class="inline-flex items-center gap-2 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 font-bold px-6 py-3 rounded-xl transition-all text-[10px] uppercase tracking-wider border border-emerald-500/20">
							<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
							Download Backup
						</a>
					</div>
					<div>
						<h4 class="text-white font-bold text-xs uppercase mb-3">Import Data</h4>
						<p class="text-zinc-500 text-[10px] mb-4">Restore your data from a previously exported .sql backup. <span class="text-app-pink font-bold">This will overwrite existing records.</span></p>
						<form action="/api/db/import" method="POST" enctype="multipart/form-data" class="flex flex-col gap-3">
							<input type="file" name="database" accept=".sql" required class="block w-full text-[10px] text-zinc-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-[10px] file:font-bold file:bg-zinc-800 file:text-zinc-300 hover:file:bg-zinc-700 cursor-pointer">
							<button type="submit" class="bg-zinc-100 hover:bg-white text-black font-bold px-6 py-2.5 rounded-xl transition-all text-[10px] uppercase tracking-wider">
								Upload & Overwrite
							</button>
						</form>
					</div>
					</div>

					<div class="mt-8 pt-6 border-t border-zinc-800/50">
					<h4 class="text-red-500 font-black text-xs uppercase mb-2">Danger Zone</h4>
					<p class="text-zinc-500 text-[10px] mb-4">Permanently delete all your workout plans, logs, and body metrics. This cannot be undone.</p>
					<button hx-post="/api/db/delete-all" hx-confirm="ARE YOU SURE? This will permanently wipe all your workout data and logs!" class="bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white border border-red-500/20 font-bold px-6 py-3 rounded-xl transition-all text-[10px] uppercase tracking-wider">
						Delete All My Data
					</button>
					</div>
					</div>


			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">Current Measurements</h3>
				</div>
				<div class="grid grid-cols-2 sm:grid-cols-5 md:grid-cols-9 gap-4">
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Age</span>
						<span class="font-display font-black text-2xl text-white">%d</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Sex</span>
						<span class="font-display font-black text-xl text-white uppercase tracking-tighter">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Height</span>
						<span class="font-display font-black text-2xl text-white">%s<span class="text-xs text-zinc-500 ml-1">cm</span></span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center border-app-yellow/40">
						<span class="block text-[10px] uppercase font-bold text-app-yellow mb-1">Goal</span>
						<span class="font-display font-black text-2xl text-white">%s<span class="text-xs text-zinc-500 ml-1">kg</span></span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Neck</span>
						<span class="font-display font-black text-2xl text-white">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Belly</span>
						<span class="font-display font-black text-2xl text-white">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Arms</span>
						<span class="font-display font-black text-2xl text-white">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Calf</span>
						<span class="font-display font-black text-2xl text-white">%s</span>
					</div>
				</div>
			</div>

			<!-- GYM Logs Section -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden mt-8">
				<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">GYM Logs</h3>
					<div class="flex items-center gap-2 bg-app-surface/80 backdrop-blur-md border border-white/5 p-1 rounded-xl relative">
						<button onclick="changeLogDate(-1)" class="p-2 text-zinc-500 hover:text-white transition-colors"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg></button>
						<span id="log-date-display" class="text-xs font-mono font-bold text-white px-2 cursor-pointer hover:text-app-pink transition-colors" onclick="toggleCalendarPopup()">%s</span>
						<button onclick="changeLogDate(1)" class="p-2 text-zinc-500 hover:text-white transition-colors"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg></button>


					</div>
				</div>

				<div class="mb-4 flex gap-2">
					<button id="btn-edit" onclick="toggleLogMode('edit')" class="px-4 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest transition-all bg-app-pink text-white shadow-lg">Edit</button>
					<button id="btn-preview" onclick="toggleLogMode('preview')" class="px-4 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest transition-all bg-app-card text-zinc-400">Preview</button>
				</div>

				<div id="log-edit-container">
					<textarea id="log-content" style="height: 600px; min-height: 600px;" class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-4 text-sm text-zinc-100 outline-none focus:border-app-pink transition-colors font-mono resize-none shadow-inner" placeholder="Type your gym notes here... (Markdown supported)"></textarea>
				</div>
				<div id="log-preview-container" style="height: 600px; min-height: 600px;" class="hidden w-full bg-zinc-900/30 border border-zinc-800/50 rounded-xl px-6 py-4 prose prose-invert prose-sm max-w-none text-zinc-100 overflow-y-auto shadow-inner">
					<!-- Preview Content -->
				</div>

				<div class="flex flex-wrap gap-3 mt-6">
					<button onclick="saveGymLog()" class="flex-1 min-w-[140px] bg-app-pink text-white font-bold px-6 py-3 rounded-xl hover:bg-app-pink/80 transition-all shadow-[0_0_12px_rgba(255,0,160,0.25)] text-[10px] uppercase tracking-wider">Save Log</button>
					<button onclick="exportCurrentLog()" class="flex-1 min-w-[140px] bg-app-card text-white font-bold px-6 py-3 rounded-xl hover:bg-zinc-700 transition-all text-[10px] uppercase tracking-wider text-center flex items-center justify-center gap-2">
						<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
						Export (.md)
					</button>
				</div>
			</div>

			<!-- Calendar Popup (Fixed Modal) -->
			<div id="calendar-popup" class="hidden fixed inset-0 z-[100] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4" onclick="toggleCalendarPopup()" style="position: fixed;">
				<div class="bg-zinc-900 border border-white/10 rounded-[2rem] p-6 sm:p-8 shadow-2xl w-full max-w-sm relative" onclick="event.stopPropagation()">
					<button onclick="toggleCalendarPopup()" class="absolute top-4 right-4 text-zinc-500 hover:text-white bg-white/5 hover:bg-white/10 rounded-full p-2 transition-colors">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
					</button>
					<div class="flex items-center justify-center gap-6 mb-6 mt-2">
						<button onclick="changeCalendarMonth(-1)" class="p-2 flex items-center justify-center text-zinc-400 hover:text-white bg-white/5 hover:bg-white/10 rounded-full transition-colors">
							<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
						</button>
						<span id="calendar-month-display" class="text-sm font-black uppercase tracking-widest text-white text-center w-32">MAY 2026</span>
						<button onclick="changeCalendarMonth(1)" class="p-2 flex items-center justify-center text-zinc-400 hover:text-white bg-white/5 hover:bg-white/10 rounded-full transition-colors">
							<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
						</button>
					</div>
					<div class="grid grid-cols-7 gap-2 mb-3">
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Su</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Mo</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Tu</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">We</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Th</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Fr</div>
						<div class="text-[10px] text-center text-zinc-500 font-bold uppercase">Sa</div>
					</div>
					<div id="calendar-days" class="grid grid-cols-7 gap-2">
						<!-- Days injected by JS -->
					</div>
				</div>
			</div>
	`, tPro, tCarb, tFat, age, gender, formatVal(height), formatVal(goalWeight), formatVal(neck), formatVal(belly), formatVal(arms), formatVal(calf), time.Now().Format("2006-01-02"))

	w.Write([]byte(html))
}
