package handlers

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"gama-fit/database"
)

func HandleMacrosSummary(w http.ResponseWriter, r *http.Request) {
	var targetCal, targetPro, targetCarb, targetFat int
	if err := database.DB.QueryRow("SELECT calories, protein, carbs, fats FROM user_macros_final WHERE id = 1").Scan(&targetCal, &targetPro, &targetCarb, &targetFat); err != nil || targetCal == 0 {
		targetCal, targetPro, targetCarb, targetFat = 2500, 200, 300, 70
	}

	var calories, protein, carbs, fats int
	_ = database.DB.QueryRow("SELECT COALESCE(SUM(calories),0), COALESCE(SUM(protein),0), COALESCE(SUM(carbs),0), COALESCE(SUM(fats),0) FROM daily_meals WHERE log_date = date('now')").Scan(&calories, &protein, &carbs, &fats)

	pctCal := (float64(calories) / float64(targetCal)) * 100
	pctPro := (float64(protein) / float64(targetPro)) * 100
	pctCarb := (float64(carbs) / float64(targetCarb)) * 100
	pctFat := (float64(fats) / float64(targetFat)) * 100

	if pctCal > 100 {
		pctCal = 100
	}
	if pctPro > 100 {
		pctPro = 100
	}
	if pctCarb > 100 {
		pctCarb = 100
	}
	if pctFat > 100 {
		pctFat = 100
	}

	calCircumference := 2 * math.Pi * 45
	calOffset := calCircumference - (pctCal / 100.0 * calCircumference)

	macroCircumference := 2 * math.Pi * 40
	proOffset := macroCircumference - (pctPro / 100.0 * macroCircumference)
	carbOffset := macroCircumference - (pctCarb / 100.0 * macroCircumference)
	fatOffset := macroCircumference - (pctFat / 100.0 * macroCircumference)

	fmt.Fprintf(w, `
		<div class="flex items-center justify-between mb-8 relative z-10">
			<h3 class="text-white font-black uppercase tracking-wider text-sm">Daily Breakdown</h3>
			<button onclick="document.getElementById('target-modal').classList.remove('hidden')" class="bg-zinc-900 border border-zinc-700 hover:border-app-yellow text-zinc-300 px-4 py-2 rounded-xl text-xs font-bold tracking-wider uppercase transition-colors flex items-center gap-2">
				<svg class="w-4 h-4 text-app-yellow" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
				Edit Targets
			</button>
		</div>

		<div class="flex flex-col md:flex-row items-center justify-between gap-8 sm:gap-12 relative z-10">
			<div class="relative flex-shrink-0">
				<div class="absolute inset-0 bg-app-yellow/10 blur-3xl rounded-full scale-90"></div>
				<div class="relative w-48 h-48 sm:w-64 sm:h-64 flex items-center justify-center">
					<svg class="w-full h-full -rotate-90 drop-shadow-[0_0_25px_rgba(251,255,0,0.5)]" viewBox="0 0 100 100">
						<circle class="text-zinc-800/40" stroke-width="5" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50"/>
						<circle class="text-app-yellow macro-ring" stroke-width="5" stroke-linecap="round" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50" data-target="%f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f; transition: stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1);"/>
					</svg>
					<div class="absolute inset-0 flex flex-col items-center justify-center text-center">
						<span class="font-display font-black text-4xl sm:text-6xl text-white drop-shadow-lg tracking-tighter">%d</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-app-yellow tracking-widest mt-1 bg-app-yellow/10 px-2 sm:px-3 py-1 rounded-full border border-app-yellow/20">/ %d kcal</span>
					</div>
				</div>
			</div>

			<div class="flex-1 w-full grid grid-cols-3 gap-3 sm:gap-4">
				<div class="bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-app-pink macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f; transition: stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1);"/>
					</svg>
						<div class="absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm">%dg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Protein</span>
				</div>

				<div class="bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-blue-500 macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f; transition: stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1);"/>
					</svg>
						<div class="absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm">%dg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Carbs</span>
				</div>

				<div class="bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-emerald-400 macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f; transition: stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1);"/>
					</svg>
						<div class="absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm">%dg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Fats</span>
				</div>
			</div>
		</div>
	`, pctCal, calCircumference, calOffset, calories, targetCal,
		pctPro, macroCircumference, proOffset, protein,
		pctCarb, macroCircumference, carbOffset, carbs,
		pctFat, macroCircumference, fatOffset, fats)

	fmt.Fprintf(w, `
	<div id="target-modal" hx-swap-oob="true" class="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-sm hidden transition-all p-4">
		<div class="bg-[#140a10] border border-zinc-800 rounded-[2rem] p-6 sm:p-8 w-full max-w-md shadow-2xl">
			<div class="flex items-center justify-between mb-6">
				<h3 class="text-white font-black uppercase tracking-wider text-sm">Set Daily Targets</h3>
				<button onclick="document.getElementById('target-modal').classList.add('hidden')" class="text-zinc-500 hover:text-white p-2">✕</button>
			</div>
			<form hx-post="/api/macros/targets" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-on::after-request="document.getElementById('target-modal').classList.add('hidden');" class="flex flex-col gap-4">
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
					<div>
						<label class="text-[10px] uppercase font-bold text-app-pink tracking-wider mb-1.5 block">Protein (g)</label>
						<input type="number" name="protein" value="%d" required class="w-full bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-pink font-mono">
					</div>
					<div>
						<label class="text-[10px] uppercase font-bold text-blue-400 tracking-wider mb-1.5 block">Carbs (g)</label>
						<input type="number" name="carbs" value="%d" required class="w-full bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-blue-500 font-mono">
					</div>
					<div>
						<label class="text-[10px] uppercase font-bold text-emerald-400 tracking-wider mb-1.5 block">Fats (g)</label>
						<input type="number" name="fats" value="%d" required class="w-full bg-zinc-900/50 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-emerald-500 font-mono">
					</div>
				</div>
				<button type="submit" class="mt-4 w-full bg-white text-black font-black py-4 rounded-xl hover:bg-zinc-200 transition-all uppercase tracking-widest text-sm">Save Targets</button>
			</form>
		</div>
	</div>
	`, targetPro, targetCarb, targetFat)
}

func HandleSetTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		pro, _ := strconv.Atoi(r.FormValue("protein"))
		carb, _ := strconv.Atoi(r.FormValue("carbs"))
		fat, _ := strconv.Atoi(r.FormValue("fats"))
		cal := pro*4 + carb*4 + fat*9

		_, _ = database.DB.Exec(`
			INSERT INTO user_macros_final (id, calories, protein, carbs, fats) 
			VALUES (1, ?, ?, ?, ?) 
			ON CONFLICT(id) DO UPDATE SET 
			calories = excluded.calories, protein = excluded.protein, carbs = excluded.carbs, fats = excluded.fats
		`, cal, pro, carb, fat)
	}
	HandleMacrosSummary(w, r)
}

func HandleMeals(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()

		if quickName := r.FormValue("quick_name"); quickName != "" {
			pro, _ := strconv.Atoi(r.FormValue("quick_protein"))
			carb, _ := strconv.Atoi(r.FormValue("quick_carbs"))
			fat, _ := strconv.Atoi(r.FormValue("quick_fats"))
			cal := pro*4 + carb*4 + fat*9

			_, _ = database.DB.Exec("INSERT INTO daily_meals (name, calories, protein, carbs, fats, log_date, log_time) VALUES (?, ?, ?, ?, ?, date('now'), time('now'))", quickName, cal, pro, carb, fat)
			fmt.Fprint(w, `<div id="daily-breakdown-updater" hx-swap-oob="true" hx-get="/api/macros/summary" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-trigger="load"></div>`)
			return
		} else if selectedFood := r.FormValue("catalog_food"); selectedFood != "" {
			grams, _ := strconv.Atoi(r.FormValue("catalog_grams"))
			if grams <= 0 {
				grams = 100
			}
			var cal, pro, carb, fat, baseWeight int
			err := database.DB.QueryRow("SELECT calories, protein, carbs, fats, weight FROM food_catalog WHERE name = ?", selectedFood).Scan(&cal, &pro, &carb, &fat, &baseWeight)
			if err == nil {
				if baseWeight <= 0 {
					baseWeight = 100
				}
				ratio := float64(grams) / float64(baseWeight)
				finalPro := int(math.Round(float64(pro) * ratio))
				finalCarb := int(math.Round(float64(carb) * ratio))
				finalFat := int(math.Round(float64(fat) * ratio))
				finalCal := finalPro*4 + finalCarb*4 + finalFat*9

				_, _ = database.DB.Exec("INSERT INTO daily_meals (name, calories, protein, carbs, fats, log_date, log_time) VALUES (?, ?, ?, ?, ?, date('now'), time('now'))", selectedFood, finalCal, finalPro, finalCarb, finalFat)
			}
		} else {
			name := r.FormValue("name")
			pro, _ := strconv.Atoi(r.FormValue("protein"))
			carb, _ := strconv.Atoi(r.FormValue("carbs"))
			fat, _ := strconv.Atoi(r.FormValue("fats"))
			weight, _ := strconv.Atoi(r.FormValue("weight"))
			if weight <= 0 {
				weight = 100
			}
			cal := pro*4 + carb*4 + fat*9

			if name != "" {
				_, _ = database.DB.Exec("INSERT INTO food_catalog (name, calories, protein, carbs, fats, weight) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(name) DO UPDATE SET calories=excluded.calories, protein=excluded.protein, carbs=excluded.carbs, fats=excluded.fats, weight=excluded.weight", name, cal, pro, carb, fat, weight)
			}
			// Only save, do NOT log automatically now.
		}
	} else if r.Method == http.MethodDelete {
		if catName := r.URL.Query().Get("catalog_name"); catName != "" {
			_, _ = database.DB.Exec("DELETE FROM food_catalog WHERE name = ?", catName)
		} else {
			id := r.URL.Query().Get("id")
			_, _ = database.DB.Exec("DELETE FROM daily_meals WHERE id = ?", id)
		}
	}

	rows, err := database.DB.Query("SELECT name, calories, protein, carbs, fats, weight FROM food_catalog ORDER BY name ASC")
	if err != nil {
		w.Write([]byte(`<option value="" disabled selected>No saved foods yet. Create one below!</option>`))
		return
	}
	defer rows.Close()

	hasCatalog := false
	catalogHtml := `<option value="" disabled selected>Select from catalogue...</option>`
	manageListHtml := ""
	for rows.Next() {
		hasCatalog = true
		var name string
		var cal, pro, carb, fat, weight int
		if err := rows.Scan(&name, &cal, &pro, &carb, &fat, &weight); err == nil {
			catalogHtml += fmt.Sprintf(`<option value="%s">%s ( %dg: %dP | %dC | %dF )</option>`, name, name, weight, pro, carb, fat)
			manageListHtml += fmt.Sprintf(`
				<div class="flex items-center justify-between bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 mb-2 group hover:border-zinc-600 transition-colors">
					<div>
						<div class="text-white text-sm font-bold">%s</div>
						<div class="text-zinc-500 text-[10px] uppercase font-bold tracking-widest">%dg serving: %dP | %dC | %dF</div>
					</div>
					<button hx-delete="/api/meals?catalog_name=%s" hx-target="#food-catalog-container" class="text-zinc-600 hover:text-red-500 transition-colors p-1">
						<svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
					</button>
				</div>
			`, name, weight, pro, carb, fat, url.QueryEscape(name))
		}
	}
	if !hasCatalog {
		catalogHtml = `<option value="" disabled selected>No saved foods yet. Create one below!</option>`
		manageListHtml = `<div class="text-zinc-600 text-center py-4 font-mono text-sm">Catalogue is empty.</div>`
	}

	fmt.Fprintf(w, `
		<form hx-post="/api/meals" hx-target="#food-catalog-container" hx-swap="innerHTML" class="flex flex-col sm:flex-row gap-3 relative z-10 w-full">
			<div class="relative flex-[2]">
				<select name="catalog_food" required class="w-full bg-zinc-900/80 border border-zinc-700 rounded-xl pl-5 pr-10 py-4 text-sm text-white outline-none focus:border-app-yellow focus:ring-2 focus:ring-app-yellow/20 transition-all appearance-none cursor-pointer font-bold shadow-lg">
					%s
				</select>
				<div class="absolute inset-y-0 right-0 flex items-center pr-4 pointer-events-none">
					<svg class="w-5 h-5 text-app-yellow drop-shadow-[0_0_5px_rgba(251,255,0,0.5)]" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
				</div>
			</div>
			<div class="relative flex-1">
				<input type="number" name="catalog_grams" placeholder="Grams" required class="w-full bg-zinc-900/80 border border-zinc-700 rounded-xl px-4 py-4 text-sm text-center text-white outline-none focus:border-app-yellow font-mono shadow-lg">
			</div>
			<button type="submit" class="bg-app-yellow text-black font-black px-8 py-4 rounded-xl hover:bg-yellow-400 transition-all shadow-[0_0_20px_rgba(251,255,0,0.3)] hover:shadow-[0_0_30px_rgba(251,255,0,0.5)] uppercase tracking-wider text-sm flex-shrink-0">Log Meal</button>
		</form>
	`, catalogHtml)

	fmt.Fprintf(w, `<div id="catalog-list" hx-swap-oob="true" class="flex flex-col gap-2 max-h-64 overflow-y-auto pr-2 scrollbar-hide">%s</div>`, manageListHtml)

	histRows, err := database.DB.Query("SELECT id, name, log_time, calories, protein, carbs, fats FROM daily_meals WHERE log_date = date('now') ORDER BY id DESC LIMIT 15")
	if err != nil {
		fmt.Fprint(w, `<div id="meals-list" hx-swap-oob="true" class="flex flex-col gap-3 max-h-96 overflow-y-auto pr-2 scrollbar-hide"><div class="text-zinc-600 text-center py-4 font-mono">No meals logged today. Get eating!</div></div>`)
		if r.Method == http.MethodPost || r.Method == http.MethodDelete {
			fmt.Fprint(w, `<div id="daily-breakdown-updater" hx-swap-oob="true" hx-get="/api/macros/summary" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-trigger="load"></div>`)
		}
		return
	}
	defer histRows.Close()

	hasHistory := false
	historyHtml := `<div id="meals-list" hx-swap-oob="true" class="flex flex-col gap-3 max-h-96 overflow-y-auto pr-2 scrollbar-hide">`
	for histRows.Next() {
		hasHistory = true
		var id, cal, pro, carb, fat int
		var name, timeStr string
		if err := histRows.Scan(&id, &name, &timeStr, &cal, &pro, &carb, &fat); err == nil {
			historyHtml += fmt.Sprintf(`
				<div class="bg-zinc-900/30 border border-zinc-800/50 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:border-zinc-700 transition-colors">
					<div>
						<h4 class="text-white font-bold text-sm">%s</h4>
						<span class="text-zinc-500 text-xs font-mono">%s</span>
					</div>
					<div class="flex gap-4 items-center">
						<div class="text-center"><span class="block text-[10px] uppercase font-bold text-zinc-500">Kcal</span><span class="text-app-yellow font-mono text-sm font-bold">%d</span></div>
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Pro</span><span class="text-app-pink font-mono text-sm font-bold">%dg</span></div>
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Carb</span><span class="text-blue-400 font-mono text-sm font-bold">%dg</span></div>
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Fat</span><span class="text-emerald-400 font-mono text-sm font-bold">%dg</span></div>
						<button hx-delete="/api/meals?id=%d" hx-target="#food-catalog-container" class="ml-2 text-zinc-600 hover:text-red-500 transition-colors"><svg class="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>
					</div>
				</div>`, name, formatTime12h(timeStr), cal, pro, carb, fat, id)
		}
	}
	if !hasHistory {
		historyHtml += `<div class="text-zinc-600 text-center py-4 font-mono">No meals logged today. Get eating!</div>`
	}
	historyHtml += `</div>`

	fmt.Fprint(w, historyHtml)

	if r.Method == http.MethodPost || r.Method == http.MethodDelete {
		fmt.Fprint(w, `<div id="daily-breakdown-updater" hx-swap-oob="true" hx-get="/api/macros/summary" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-trigger="load"></div>`)
	}
}
