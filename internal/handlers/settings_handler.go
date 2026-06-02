package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"gama-fit/database"
)

func GetPomoSettings(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	var pomo, short, long int
	err := database.DB.QueryRow("SELECT pomo_duration, short_break, long_break FROM user_stats WHERE user_id = $1", userID).Scan(&pomo, &short, &long)
	if err != nil {
		pomo, short, long = 25, 5, 15
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"pomo": %d, "short": %d, "long": %d}`, pomo, short, long)
}

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
		if r.FormValue("action") == "pomo" {
			pomo, _ := strconv.Atoi(r.FormValue("pomo_duration"))
			short, _ := strconv.Atoi(r.FormValue("short_break"))
			long, _ := strconv.Atoi(r.FormValue("long_break"))
			_, _ = database.DB.Exec("UPDATE user_stats SET pomo_duration = $1, short_break = $2, long_break = $3 WHERE user_id = $4", pomo, short, long, userID)
			w.Header().Set("HX-Trigger", "pomoUpdated")
		} else if r.FormValue("action") == "macros" {
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
					if metric == "bmi" || metric == "height" || metric == "neck" || metric == "belly" || metric == "arms" || metric == "calf" || metric == "age" {
						query := fmt.Sprintf("UPDATE user_stats SET %s = $1 WHERE user_id = $2", metric)
						_, _ = database.DB.Exec(query, val, userID)
					}
				}
			}
			w.Header().Set("HX-Trigger", "metricUpdated")
		}
	}

	var bmi, height, neck, belly, arms, calf float64
	var age int
	var gender, theme string
	var pomoDuration, shortBreak, longBreak int
	err := database.DB.QueryRow("SELECT bmi, height, neck, belly, arms, calf, age, gender, theme, pomo_duration, short_break, long_break FROM user_stats WHERE user_id = $1", userID).Scan(&bmi, &height, &neck, &belly, &arms, &calf, &age, &gender, &theme, &pomoDuration, &shortBreak, &longBreak)
	if err != nil {
		bmi, height, neck, belly, arms, calf = 0, 0, 0, 0, 0, 0
		age = 25
		gender = "male"
		theme = "default"
		pomoDuration, shortBreak, longBreak = 25, 5, 15
	}
	if gender == "" { gender = "male" }

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

			<!-- Pomodoro Settings -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden border-blue-500/20">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-blue-400 font-black uppercase tracking-wider text-sm">Pomodoro Timer</h3>
				</div>
				<form hx-post="/api/settings" hx-target="#settings-container" hx-swap="innerHTML" class="space-y-6">
					<input type="hidden" name="action" value="pomo">
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
						<button type="button" onclick="setPomo(20, 5, 15)" class="p-4 rounded-2xl border border-white/5 bg-app-surface hover:border-blue-500 transition-all text-left">
							<span class="block text-xs font-bold text-white mb-1">Popular</span>
							<span class="text-[10px] text-zinc-500 uppercase tracking-tight">20m / 5m / 15m</span>
						</button>
						<button type="button" onclick="setPomo(40, 8, 20)" class="p-4 rounded-2xl border border-white/5 bg-app-surface hover:border-blue-500 transition-all text-left">
							<span class="block text-xs font-bold text-white mb-1">Medium</span>
							<span class="text-[10px] text-zinc-500 uppercase tracking-tight">40m / 8m / 20m</span>
						</button>
						<button type="button" onclick="setPomo(60, 10, 25)" class="p-4 rounded-2xl border border-white/5 bg-app-surface hover:border-blue-500 transition-all text-left">
							<span class="block text-xs font-bold text-white mb-1">Extended</span>
							<span class="text-[10px] text-zinc-500 uppercase tracking-tight">60m / 10m / 25m</span>
						</button>
					</div>

					<div class="space-y-4">
						<div>
							<div class="flex justify-between mb-2"><label class="text-[10px] uppercase font-bold text-zinc-500">Pomodoro (min)</label><span id="pomo-val" class="text-xs font-mono text-blue-400 font-bold">%d</span></div>
							<input type="range" name="pomo_duration" min="1" max="90" value="%d" oninput="document.getElementById('pomo-val').innerText = this.value" class="w-full h-1.5 bg-zinc-800 rounded-lg appearance-none cursor-pointer accent-blue-500">
						</div>
						<div class="grid grid-cols-2 gap-4">
							<div>
								<div class="flex justify-between mb-2"><label class="text-[10px] uppercase font-bold text-zinc-500">Short Break</label><span id="short-val" class="text-xs font-mono text-blue-400 font-bold">%d</span></div>
								<input type="range" name="short_break" min="1" max="30" value="%d" oninput="document.getElementById('short-val').innerText = this.value" class="w-full h-1.5 bg-zinc-800 rounded-lg appearance-none cursor-pointer accent-blue-500">
							</div>
							<div>
								<div class="flex justify-between mb-2"><label class="text-[10px] uppercase font-bold text-zinc-500">Long Break</label><span id="long-val" class="text-xs font-mono text-blue-400 font-bold">%d</span></div>
								<input type="range" name="long_break" min="1" max="60" value="%d" oninput="document.getElementById('long-val').innerText = this.value" class="w-full h-1.5 bg-zinc-800 rounded-lg appearance-none cursor-pointer accent-blue-500">
							</div>
						</div>
					</div>
					<button type="submit" class="w-full bg-blue-500 text-white font-black py-4 rounded-xl hover:bg-blue-400 transition-all uppercase tracking-widest text-xs shadow-lg shadow-blue-500/20">Update Timer Settings</button>
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
					<button onclick="setTheme('synthwave84')" class="p-3 rounded-xl border border-white/5 bg-[#05010a] hover:border-[#ff2fa8] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Synthwave 84</button>
					<button onclick="setTheme('materialui')" class="p-3 rounded-xl border border-white/5 bg-[#0b0911] hover:border-[#ff2fa8] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Material UI</button>
					<button onclick="setTheme('gruvbox')" class="p-3 rounded-xl border border-white/5 bg-[#282828] hover:border-[#fb4934] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Gruvbox</button>
					<button onclick="setTheme('catppuccin')" class="p-3 rounded-xl border border-white/5 bg-[#1e1e2e] hover:border-[#f5c2e7] transition-all text-[10px] font-bold uppercase tracking-tight text-zinc-400 hover:text-white">Catppuccin</button>
				</div>
			</div>

			<!-- GYM Logs Section -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">GYM Logs</h3>
					<div class="flex items-center gap-2 bg-app-surface/80 backdrop-blur-md border border-white/5 p-1 rounded-xl">
						<button onclick="changeLogDate(-1)" class="p-2 text-zinc-500 hover:text-white transition-colors"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg></button>
						<span id="log-date-display" class="text-xs font-mono font-bold text-white px-2">2026-05-29</span>
						<button onclick="changeLogDate(1)" class="p-2 text-zinc-500 hover:text-white transition-colors"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg></button>
					</div>
				</div>

				<div class="mb-4 flex gap-2">
					<button id="btn-edit" onclick="toggleLogMode('edit')" class="px-4 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest transition-all bg-app-pink text-white shadow-lg">Edit</button>
					<button id="btn-preview" onclick="toggleLogMode('preview')" class="px-4 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest transition-all bg-app-card text-zinc-400">Preview</button>
				</div>

				<div id="log-edit-container">
					<textarea id="log-content" class="w-full h-64 bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-zinc-100 outline-none focus:border-app-pink transition-colors font-mono resize-none" placeholder="Type your gym notes here... (Markdown supported)"></textarea>
				</div>
				<div id="log-preview-container" class="hidden w-full min-h-[16rem] bg-zinc-900/30 border border-zinc-800/50 rounded-xl px-4 py-3 prose prose-invert prose-sm max-w-none text-zinc-100">
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

			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">Log Body Metrics</h3>
				</div>
				<form hx-post="/api/settings" hx-target="#settings-container" hx-swap="innerHTML" class="space-y-4">
					<div>
						<label class="block text-[10px] font-bold uppercase tracking-wider text-zinc-500 mb-2">Metric Type</label>
						<select name="metric" class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 transition-colors">
							<option value="age">Age (years)</option>
							<option value="gender">Gender</option>
							<option value="bmi">BMI</option>
							<option value="height">Height (cm)</option>
							<option value="neck">Neck</option>
							<option value="belly">Belly</option>
							<option value="arms">Arms</option>
							<option value="calf">Calf</option>
						</select>
						</div>
						<div>
						<label class="block text-[10px] font-bold uppercase tracking-wider text-zinc-500 mb-2">Value / Option</label>
						<input type="text" name="value" placeholder="e.g. 25 or male" required class="w-full bg-app-surface/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 transition-colors font-mono">
						</div>
					<button type="submit" class="w-full bg-blue-500 text-white font-bold px-6 py-3 rounded-xl hover:bg-blue-400 transition-all shadow-[0_0_12px_rgba(59,130,246,0.25)] text-xs uppercase tracking-wider">Save Metric</button>
				</form>
			</div>

			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-white font-black uppercase tracking-wider text-sm">Current Measurements</h3>
				</div>
				<div class="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-8 gap-4">
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Age</span>
						<span class="font-display font-black text-2xl text-white">%d</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Gender</span>
						<span class="font-display font-black text-xl text-white uppercase tracking-tighter">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">BMI</span>
						<span class="font-display font-black text-2xl text-white">%s</span>
					</div>
					<div class="bg-app-surface/50 border border-white/5 rounded-xl p-4 text-center">
						<span class="block text-[10px] uppercase font-bold text-zinc-500 mb-1">Height</span>
						<span class="font-display font-black text-2xl text-white">%s<span class="text-xs text-zinc-500 ml-1">cm</span></span>
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

			<!-- DB Backup Actions at Bottom -->
			<div class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden border-app-yellow/20">
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-app-yellow font-black uppercase tracking-wider text-sm">Database Management</h3>
				</div>
				<div class="flex flex-col sm:flex-row gap-4">
					<a href="/api/db/export" class="flex-1 flex items-center justify-center gap-2 bg-app-surface/80 backdrop-blur-md border border-white/5 hover:border-app-yellow/50 text-zinc-400 hover:text-app-yellow px-4 py-4 rounded-xl text-xs font-bold uppercase tracking-widest transition-all shadow-xl group">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
						<span>Export DB</span>
					</a>

					<button onclick="document.getElementById('db-import-input').click()" class="flex-1 flex items-center justify-center gap-2 bg-app-surface/80 backdrop-blur-md border border-white/5 hover:border-app-pink/50 text-zinc-400 hover:text-app-pink px-4 py-4 rounded-xl text-xs font-bold uppercase tracking-widest transition-all shadow-xl group">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
						<span>Import DB</span>
					</button>
					<form id="db-import-form" action="/api/db/import" method="POST" enctype="multipart/form-data" class="hidden">
						<input type="file" id="db-import-input" name="database" onchange="if(confirm('Importing will overwrite current data. Continue?')) document.getElementById('db-import-form').submit()">
					</form>
				</div>
				<p class="text-[10px] text-zinc-500 mt-4 text-center uppercase tracking-widest font-bold">Note: PostgreSQL exports are currently handled via system tools.</p>
			</div>
		</div>

		<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
		<script>
			function setPomo(p, s, l) {
				const form = document.querySelector('form[hx-post="/api/settings"] input[name="action"][value="pomo"]').closest('form');
				form.querySelector('input[name="pomo_duration"]').value = p;
				form.querySelector('input[name="short_break"]').value = s;
				form.querySelector('input[name="long_break"]').value = l;
				document.getElementById('pomo-val').innerText = p;
				document.getElementById('short-val').innerText = s;
				document.getElementById('long-val').innerText = l;
			}

			let currentLogDate = new Date().toISOString().split('T')[0];
			
			function updateDateDisplay() {
				document.getElementById('log-date-display').innerText = currentLogDate;
				fetchLog();
			}

			function changeLogDate(days) {
				let d = new Date(currentLogDate);
				d.setDate(d.getDate() + days);
				currentLogDate = d.toISOString().split('T')[0];
				updateDateDisplay();
			}

			async function fetchLog() {
				const res = await fetch('/api/logs?date=' + currentLogDate);
				const content = await res.text();
				document.getElementById('log-content').value = content;
				if (document.getElementById('log-preview-container').classList.contains('hidden') === false) {
					renderPreview();
				}
			}

			async function saveGymLog() {
				const content = document.getElementById('log-content').value;
				const formData = new FormData();
				formData.append('date', currentLogDate);
				formData.append('content', content);
				
				const res = await fetch('/api/logs', {
					method: 'POST',
					body: formData
				});
				
				if (res.ok) {
					alert('Saved!');
				} else {
					alert('Error saving log');
				}
			}

			function toggleLogMode(mode) {
				const edit = document.getElementById('log-edit-container');
				const preview = document.getElementById('log-preview-container');
				const btnEdit = document.getElementById('btn-edit');
				const btnPreview = document.getElementById('btn-preview');
				
				if (mode === 'edit') {
					edit.classList.remove('hidden');
					preview.classList.add('hidden');
					btnEdit.classList.add('bg-app-pink', 'text-white');
					btnEdit.classList.remove('bg-app-card', 'text-zinc-400');
					btnPreview.classList.add('bg-app-card', 'text-zinc-400');
					btnPreview.classList.remove('bg-app-pink', 'text-white');
				} else {
					edit.classList.add('hidden');
					preview.classList.remove('hidden');
					btnPreview.classList.add('bg-app-pink', 'text-white');
					btnPreview.classList.remove('bg-app-card', 'text-zinc-400');
					btnEdit.classList.add('bg-app-card', 'text-zinc-400');
					btnEdit.classList.remove('bg-app-pink', 'text-white');
					renderPreview();
				}
			}

			function renderPreview() {
				const content = document.getElementById('log-content').value;
				document.getElementById('log-preview-container').innerHTML = marked.parse(content);
			}

			function exportCurrentLog() {
				const content = document.getElementById('log-content').value;
				const blob = new Blob([content], { type: 'text/markdown' });
				const url = window.URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = "gym_log_" + currentLogDate + ".md";
				a.click();
				window.URL.revokeObjectURL(url);
			}

			async function setTheme(theme) {
				const formData = new FormData();
				formData.append('theme', theme);
				await fetch('/api/settings/theme', {
					method: 'POST',
					body: formData
				});
				applyTheme(theme);
			}

			function applyTheme(theme) {
				document.documentElement.setAttribute('data-theme', theme);
				localStorage.setItem('app-theme', theme);
			}

			// Initialize
			updateDateDisplay();
			const savedTheme = localStorage.getItem('app-theme') || 'default';
			applyTheme(savedTheme);
		</script>
	`, tPro, tCarb, tFat, pomoDuration, pomoDuration, shortBreak, shortBreak, longBreak, longBreak, age, gender, formatVal(bmi), formatVal(height), formatVal(neck), formatVal(belly), formatVal(arms), formatVal(calf))

	w.Write([]byte(html))
}
