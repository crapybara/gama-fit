package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"time"

	"gama-fit/database"
)

func HandleMacrosSummary(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, _ := getLocalTime(r)

	// 1. Fetch User Data for BMR & Targets
	var targetCal int
	var targetPro, targetCarb, targetFat float64
	var age int
	var gender string
	var height, goalWeight float64
	_ = database.DB.QueryRow("SELECT calories, protein, carbs, fats FROM user_macros_final WHERE user_id = $1", userID).Scan(&targetCal, &targetPro, &targetCarb, &targetFat)
	_ = database.DB.QueryRow("SELECT age, gender, height, goal_weight FROM user_stats WHERE user_id = $1", userID).Scan(&age, &gender, &height, &goalWeight)

	if targetCal == 0 {
		targetCal, targetPro, targetCarb, targetFat = 2500, 200, 300, 70
	}
	if age <= 0 {
		age = 25
	}
	if gender == "" {
		gender = "male"
	}
	if height <= 0 {
		height = 175
	}

	// 2. Fetch Today's Weight
	var weight float64
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 AND log_date = $2", userID, localDate).Scan(&weight)
	if weight <= 0 {
		weight = 75
	}

	// 3. Calculate BMR (Mifflin-St Jeor)
	bmr := 10*weight + 6.25*height - 5*float64(age)
	if gender == "male" {
		bmr += 5
	} else {
		bmr -= 161
	}

	// 4. Calculate Calories Consumed
	var calories int
	var protein, carbs, fats float64
	_ = database.DB.QueryRow("SELECT COALESCE(SUM(calories),0), COALESCE(SUM(protein),0), COALESCE(SUM(carbs),0), COALESCE(SUM(fats),0) FROM daily_meals WHERE user_id = $1 AND log_date = $2", userID, localDate).Scan(&calories, &protein, &carbs, &fats)

	// 5. Calculate Active Calories Burnt
	activeBurn := 0.0
	// Cardio Logs
	rows, err := database.DB.Query("SELECT intensity, duration FROM cardio_logs WHERE user_id = $1 AND logged_date = $2", userID, localDate)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var intensity string
			var duration int
			if err := rows.Scan(&intensity, &duration); err == nil {
				met := 6.0 // Default Moderate
				switch intensity {
				case "Light":
					met = 3.5
				case "Vigorous":
					met = 9.0
				case "Very Hard":
					met = 12.0
				}
				activeBurn += (met * weight * (float64(duration) / 60.0))
			}
		}
	}
	// Freestyle Logs (simplified: 5 kcal per set)
	var freestyleSets int
	_ = database.DB.QueryRow("SELECT COUNT(*) FROM freestyle_logs WHERE user_id = $1 AND logged_date = $2", userID, localDate).Scan(&freestyleSets)
	activeBurn += float64(freestyleSets) * 5.0

	// 6. Percentages for Rings
	pctCal := (float64(calories) / float64(targetCal)) * 100
	pctBMR := (bmr / 2500.0) * 100 // Scale relative to a baseline or itself
	if pctBMR > 100 {
		pctBMR = 100
	}

	pctBurn := (activeBurn / 1000.0) * 100 // Scale relative to a daily active goal
	if pctBurn > 100 {
		pctBurn = 100
	}

	if pctCal > 100 {
		pctCal = 100
	}

	pctPro := (protein / targetPro) * 100
	pctCarb := (carbs / targetCarb) * 100
	pctFat := (fats / targetFat) * 100
	if pctPro > 100 {
		pctPro = 100
	}
	if pctCarb > 100 {
		pctCarb = 100
	}
	if pctFat > 100 {
		pctFat = 100
	}

	// 7. SVG Dimensions
	c1 := 2 * math.Pi * 45.0 // Consumed (Outer)
	c2 := 2 * math.Pi * 37.0 // BMR (Middle)
	c3 := 2 * math.Pi * 29.0 // Burned (Inner)

	off1 := c1 - (pctCal / 100.0 * c1)
	off2 := c2 - (pctBMR / 100.0 * c2)
	off3 := c3 - (pctBurn / 100.0 * c3)

	macroCircumference := 2 * math.Pi * 40
	proOffset := macroCircumference - (pctPro / 100.0 * macroCircumference)
	carbOffset := macroCircumference - (pctCarb / 100.0 * macroCircumference)
	fatOffset := macroCircumference - (pctFat / 100.0 * macroCircumference)

	// 8. Weight Prediction
	predictionHTML := ""
	if goalWeight > 0 {
		_, _, _, hasEnoughData, statusMsg := CalculateWeightTrajectory(userID)

		textColor := "text-white"
		if !hasEnoughData {
			textColor = "text-zinc-500" // Muted gray for "Not enough data"
		}

		predictionHTML = fmt.Sprintf(`
			<div class="mt-8 border-t border-white/5 pt-6 relative z-10 w-full flex flex-col items-center justify-center">
				<p class="text-zinc-500 text-[10px] uppercase font-black tracking-[0.2em] mb-2">Weight Trajectory</p>
				<p class="%s text-sm font-bold bg-zinc-900/80 px-4 py-2 rounded-xl border border-white/5 shadow-lg">%s</p>
			</div>
		`, textColor, statusMsg)
	}

	fmt.Fprintf(w, `
		<div class="flex items-center justify-between mb-8 relative z-10">
			<h3 class="text-white font-black uppercase tracking-wider text-sm">Energy Balance</h3>
			<div class="flex gap-4">
				<div class="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity" onmouseenter="setCalView('consumed')" onmouseleave="setCalView('default')">
					<div class="w-3 h-3 rounded-full bg-app-yellow"></div>
					<span class="text-[10px] font-bold text-zinc-500 uppercase">Consumed</span>
				</div>
				<div class="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity" onmouseenter="setCalView('bmr')" onmouseleave="setCalView('default')">
					<div class="w-3 h-3 rounded-full bg-blue-500"></div>
					<span class="text-[10px] font-bold text-zinc-500 uppercase">BMR</span>
				</div>
				<div class="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity" onmouseenter="setCalView('active')" onmouseleave="setCalView('default')">
					<div class="w-3 h-3 rounded-full bg-emerald-500"></div>
					<span class="text-[10px] font-bold text-zinc-500 uppercase">Active</span>
				</div>
			</div>
		</div>

		<div class="flex flex-col md:flex-row items-center justify-between gap-8 sm:gap-12 relative z-10">
			<div class="relative flex-shrink-0">
				<div class="absolute inset-0 bg-app-yellow/10 blur-3xl rounded-full scale-90 transition-all duration-500" id="ring-glow"></div>
				<div class="relative w-48 h-48 sm:w-64 sm:h-64 flex items-center justify-center">
					<svg class="w-full h-full -rotate-90 drop-shadow-[0_0_20px_rgba(0,0,0,0.5)]" viewBox="0 0 100 100">
						<!-- Consumed Ring (Outer) -->
						<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50"/>
						<circle class="text-app-yellow macro-ring cursor-pointer" 
								onmouseenter="setCalView('consumed')" onmouseleave="setCalView('default')"
								stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="45" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;">
						</circle>
						
						<!-- BMR Ring (Middle) -->
						<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="37" cx="50" cy="50"/>
						<circle class="text-blue-500 macro-ring cursor-pointer" 
								onmouseenter="setCalView('bmr')" onmouseleave="setCalView('default')"
								stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="37" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;">
						</circle>

						<!-- Burned Ring (Inner) -->
						<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="29" cx="50" cy="50"/>
						<circle class="text-emerald-500 macro-ring cursor-pointer" 
								onmouseenter="setCalView('active')" onmouseleave="setCalView('default')"
								stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="29" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;">
						</circle>
					</svg>

					<!-- Consumed View (Default) -->
					<div id="view-consumed" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-100 scale-100">
						<span class="font-display font-black text-4xl sm:text-6xl text-white drop-shadow-lg tracking-tighter">%d</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-app-yellow tracking-widest mt-1 bg-app-yellow/10 px-2 sm:px-3 py-1 rounded-full border border-app-yellow/20">Calories In</span>
					</div>

					<!-- BMR View -->
					<div id="view-bmr" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-blue-400 drop-shadow-lg tracking-tighter">%.0f</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-blue-400 tracking-widest mt-1 bg-blue-500/10 px-2 sm:px-3 py-1 rounded-full border border-blue-500/20">BMR kcal</span>
					</div>

					<!-- Active View -->
					<div id="view-active" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-emerald-400 drop-shadow-lg tracking-tighter">%.0f</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-emerald-400 tracking-widest mt-1 bg-emerald-500/10 px-2 sm:px-3 py-1 rounded-full border border-emerald-500/20">Active Burn</span>
					</div>

					<!-- Total Out View -->
					<div id="view-total" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-white drop-shadow-lg tracking-tighter">%.0f</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-zinc-400 tracking-widest mt-1 bg-zinc-900/50 px-2 sm:px-3 py-1 rounded-full border border-white/10">Total Out</span>
					</div>

					<!-- Protein View -->
					<div id="view-protein" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-app-pink drop-shadow-lg tracking-tighter">%.1fg</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-app-pink tracking-widest mt-1 bg-app-pink/10 px-2 sm:px-3 py-1 rounded-full border border-app-pink/20">Protein Target: %.1fg</span>
					</div>

					<!-- Carbs View -->
					<div id="view-carbs" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-blue-400 drop-shadow-lg tracking-tighter">%.1fg</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-blue-400 tracking-widest mt-1 bg-blue-500/10 px-2 sm:px-3 py-1 rounded-full border border-blue-500/20">Carbs Target: %.1fg</span>
					</div>

					<!-- Fats View -->
					<div id="view-fats" class="absolute inset-0 flex flex-col items-center justify-center text-center transition-all duration-300 opacity-0 scale-90 pointer-events-none">
						<span class="font-display font-black text-4xl sm:text-6xl text-emerald-400 drop-shadow-lg tracking-tighter">%.1fg</span>
						<span class="text-[9px] sm:text-[10px] uppercase font-bold text-emerald-400 tracking-widest mt-1 bg-emerald-500/10 px-2 sm:px-3 py-1 rounded-full border border-emerald-500/20">Fats Target: %.1fg</span>
					</div>
				</div>
			</div>

			<script>
				function setCalView(view) {
					const views = ['consumed', 'bmr', 'active', 'total', 'protein', 'carbs', 'fats'];
					const glow = document.getElementById('ring-glow');
					
					views.forEach(v => {
						const el = document.getElementById('view-' + v);
						if(!el) return;
						
						if(v === view || (view === 'default' && v === 'consumed')) {
							el.classList.remove('opacity-0', 'scale-90', 'pointer-events-none');
							el.classList.add('opacity-100', 'scale-100');
						} else {
							el.classList.remove('opacity-100', 'scale-100');
							el.classList.add('opacity-0', 'scale-90', 'pointer-events-none');
						}
					});

					if(glow) {
						if(view === 'bmr') glow.className = "absolute inset-0 bg-blue-500/10 blur-3xl rounded-full scale-90 transition-all duration-500";
						else if(view === 'active') glow.className = "absolute inset-0 bg-emerald-500/10 blur-3xl rounded-full scale-90 transition-all duration-500";
						else if(view === 'protein') glow.className = "absolute inset-0 bg-app-pink/10 blur-3xl rounded-full scale-90 transition-all duration-500";
						else if(view === 'carbs') glow.className = "absolute inset-0 bg-blue-500/10 blur-3xl rounded-full scale-90 transition-all duration-500";
						else if(view === 'fats') glow.className = "absolute inset-0 bg-emerald-500/10 blur-3xl rounded-full scale-90 transition-all duration-500";
						else glow.className = "absolute inset-0 bg-app-yellow/10 blur-3xl rounded-full scale-90 transition-all duration-500";
					}
				}
			</script>

			<div class="flex-1 w-full grid grid-cols-3 gap-3 sm:gap-4">
				<div class="macro-box group bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300"
				     onmouseenter="setCalView('protein')" onmouseleave="setCalView('default')">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-app-pink macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;"/>
					</svg>
						<div class="macro-grams absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm transition-opacity">%.1fg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Protein</span>
				</div>

				<div class="macro-box group bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300"
				     onmouseenter="setCalView('carbs')" onmouseleave="setCalView('default')">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-blue-500 macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;"/>
					</svg>
						<div class="macro-grams absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm transition-opacity">%.1fg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Carbs</span>
				</div>

				<div class="macro-box group bg-zinc-900/50 border border-white/5 rounded-2xl sm:rounded-[1.5rem] p-3 sm:p-5 flex flex-col items-center justify-center hover:bg-zinc-900/80 transition-all duration-300"
				     onmouseenter="setCalView('fats')" onmouseleave="setCalView('default')">
					<div class="relative w-14 h-14 sm:w-20 sm:h-20 mb-2 sm:mb-3">
						<svg class="w-full h-full -rotate-90" viewBox="0 0 100 100">
							<circle class="text-zinc-800/40" stroke-width="6" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50"/>
							<circle class="text-emerald-400 macro-ring" stroke-width="6" stroke-linecap="round" stroke="currentColor" fill="transparent" r="40" cx="50" cy="50" data-target="%.1f" style="stroke-dasharray: %.2f; stroke-dashoffset: %.2f;"/>
					</svg>
						<div class="macro-grams absolute inset-0 flex items-center justify-center text-white font-bold text-[10px] sm:text-sm transition-opacity">%.1fg</div>
					</div>
					<span class="text-zinc-500 text-[8px] sm:text-[10px] font-bold uppercase tracking-widest">Fats</span>
				</div>
			</div>
		</div>
		%s
	`, pctCal, c1, off1,
		pctBMR, c2, off2,
		pctBurn, c3, off3,
		calories, bmr, activeBurn, bmr+activeBurn,
		protein, targetPro, carbs, targetCarb, fats, targetFat,
		pctPro, macroCircumference, proOffset, protein,
		pctCarb, macroCircumference, carbOffset, carbs,
		pctFat, macroCircumference, fatOffset, fats,
		predictionHTML)
}

func HandleSetTargets(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
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
	}
	// This might be called from Settings now, so we need to return something appropriate.
	// If it's HX-Post from Settings, we should probably return a success message or nothing.
	w.WriteHeader(http.StatusOK)
}

func HandleMeals(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	localDate, localTime := getLocalTime(r)

	if r.Method == http.MethodPost {
		_ = r.ParseForm()

		qName := r.FormValue("quick_name")
		qPro := r.FormValue("quick_protein")
		qCarb := r.FormValue("quick_carbs")
		qFat := r.FormValue("quick_fats")
		qCal := r.FormValue("quick_calories")

		if qName != "" || qPro != "" || qCarb != "" || qFat != "" || qCal != "" {
			if qName == "" {
				qName = "Quick Log"
			}
			pro, _ := strconv.ParseFloat(qPro, 64)
			carb, _ := strconv.ParseFloat(qCarb, 64)
			fat, _ := strconv.ParseFloat(qFat, 64)
			calories, _ := strconv.Atoi(qCal)

			cal := int(math.Round(pro*4 + carb*4 + fat*9))
			if calories > 0 {
				cal = calories
			}

			_, _ = database.DB.Exec("INSERT INTO daily_meals (user_id, name, calories, protein, carbs, fats, log_date, log_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", userID, qName, cal, pro, carb, fat, localDate, localTime)
			fmt.Fprint(w, `<div id="daily-breakdown-updater" hx-swap-oob="true" hx-get="/api/macros/summary" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-trigger="load"></div>`)
			return
		} else if selectedFood := r.FormValue("catalog_food"); selectedFood != "" {
			grams, _ := strconv.ParseFloat(r.FormValue("catalog_grams"), 64)
			if grams <= 0 {
				grams = 100
			}
			var cal int
			var pro, carb, fat, baseWeight float64
			err := database.DB.QueryRow("SELECT calories, protein, carbs, fats, weight FROM food_catalog WHERE user_id = $1 AND name = $2", userID, selectedFood).Scan(&cal, &pro, &carb, &fat, &baseWeight)
			if err == nil {
				if baseWeight <= 0 {
					baseWeight = 100
				}
				ratio := grams / baseWeight
				finalPro := pro * ratio
				finalCarb := carb * ratio
				finalFat := fat * ratio
				finalCal := int(math.Round(float64(cal) * ratio))

				_, _ = database.DB.Exec("INSERT INTO daily_meals (user_id, name, calories, protein, carbs, fats, log_date, log_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", userID, selectedFood, finalCal, finalPro, finalCarb, finalFat, localDate, localTime)
			}
		} else {
			name := r.FormValue("name")
			pro, _ := strconv.ParseFloat(r.FormValue("protein"), 64)
			carb, _ := strconv.ParseFloat(r.FormValue("carbs"), 64)
			fat, _ := strconv.ParseFloat(r.FormValue("fats"), 64)
			weight, _ := strconv.ParseFloat(r.FormValue("weight"), 64)
			if weight <= 0 {
				weight = 100
			}
			cal := int(pro*4 + carb*4 + fat*9)

			if name != "" {
				_, _ = database.DB.Exec("INSERT INTO food_catalog (user_id, name, calories, protein, carbs, fats, weight) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT(user_id, name) DO UPDATE SET calories=excluded.calories, protein=excluded.protein, carbs=excluded.carbs, fats=excluded.fats, weight=excluded.weight", userID, name, cal, pro, carb, fat, weight)
			}
		}
	} else if r.Method == http.MethodDelete {
		if catName := r.URL.Query().Get("catalog_name"); catName != "" {
			_, _ = database.DB.Exec("DELETE FROM food_catalog WHERE user_id = $1 AND name = $2", userID, catName)
		} else {
			id := r.URL.Query().Get("id")
			_, _ = database.DB.Exec("DELETE FROM daily_meals WHERE user_id = $1 AND id = $2", userID, id)
		}
	}

	rows, err := database.DB.Query("SELECT name, calories, protein, carbs, fats, weight FROM food_catalog WHERE user_id = $1 ORDER BY name ASC", userID)
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
		var cal int
		var pro, carb, fat, weight float64
		if err := rows.Scan(&name, &cal, &pro, &carb, &fat, &weight); err == nil {
			catalogHtml += fmt.Sprintf(`<option value="%s">%s ( %d kcal | %.1fg: %.1fP | %.1fC | %.1fF )</option>`, name, name, cal, weight, pro, carb, fat)
			manageListHtml += fmt.Sprintf(`
				<div class="flex items-center justify-between bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 mb-2 group hover:border-zinc-600 transition-colors">
					<div>
						<div class="text-white text-sm font-bold">%s</div>
						<div class="text-zinc-500 text-[10px] uppercase font-bold tracking-widest">%d kcal | %.1fg serving: %.1fP | %.1fC | %.1fF</div>
					</div>
					<button hx-delete="/api/meals?catalog_name=%s" hx-target="#food-catalog-container" class="text-zinc-600 hover:text-red-500 transition-colors p-1">
						<svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
					</button>
				</div>
			`, name, cal, weight, pro, carb, fat, url.QueryEscape(name))
		}
	}
	if !hasCatalog {
		catalogHtml = `<option value="" disabled selected>No saved foods yet. Create one below!</option>`
		manageListHtml = `<div class="text-zinc-600 text-center py-4 font-mono text-sm">Catalogue is empty.</div>`
	}

	fmt.Fprintf(w, `
		<form hx-post="/api/meals" hx-target="#food-catalog-container" hx-swap="innerHTML" 
              hx-vals='js:{local_date: new Date().toISOString().split("T")[0], local_time: new Date().toTimeString().split(" ")[0]}'
              class="flex flex-col sm:flex-row gap-3 relative z-10 w-full">
			<div class="relative flex-[2]">
				<select name="catalog_food" required class="w-full bg-zinc-900/80 border border-zinc-700 rounded-xl pl-5 pr-10 py-4 text-sm text-white outline-none focus:border-app-yellow focus:ring-2 focus:ring-app-yellow/20 transition-all appearance-none cursor-pointer font-bold shadow-lg">
					%s
				</select>
				<div class="absolute inset-y-0 right-0 flex items-center pr-4 pointer-events-none">
					<svg class="w-5 h-5 text-app-yellow drop-shadow-[0_0_5px_rgba(251,255,0,0.5)]" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
				</div>
			</div>
			<div class="relative flex-1">
				<input type="number" step="0.1" name="catalog_grams" placeholder="Grams" required class="w-full bg-zinc-900/80 border border-zinc-700 rounded-xl px-4 py-4 text-sm text-center text-white outline-none focus:border-app-yellow font-mono shadow-lg">
			</div>
			<button type="submit" class="bg-app-yellow text-black font-black px-8 py-4 rounded-xl hover:bg-yellow-400 transition-all shadow-[0_0_20px_rgba(251,255,0,0.3)]  uppercase tracking-wider text-sm flex-shrink-0">Log Meal</button>
		</form>
	`, catalogHtml)

	fmt.Fprintf(w, `<div id="catalog-list" hx-swap-oob="true" class="flex flex-col gap-2 max-h-64 overflow-y-auto pr-2 scrollbar-hide">%s</div>`, manageListHtml)

	histRows, err := database.DB.Query("SELECT id, name, log_time, calories, protein, carbs, fats FROM daily_meals WHERE user_id = $1 AND log_date = $2 ORDER BY id DESC LIMIT 15", userID, localDate)
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
		var id, cal int
		var pro, carb, fat float64
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
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Pro</span><span class="text-app-pink font-mono text-sm font-bold">%.1fg</span></div>
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Carb</span><span class="text-blue-400 font-mono text-sm font-bold">%.1fg</span></div>
						<div class="text-center hidden sm:block"><span class="block text-[10px] uppercase font-bold text-zinc-500">Fat</span><span class="text-emerald-400 font-mono text-sm font-bold">%.1fg</span></div>
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
		fmt.Fprintf(w, `<div id="daily-breakdown-updater" hx-swap-oob="true" hx-get="/api/macros/summary?local_date=%s" hx-target="#daily-breakdown" hx-swap="innerHTML" hx-trigger="load"></div>`, localDate)
	}
}

type CatalogFoodItem struct {
	Name     string  `json:"name"`
	Calories int     `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fats     float64 `json:"fats"`
	Weight   float64 `json:"weight"`
}

func HandleFoodCatalogJSON(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query("SELECT name, calories, protein, carbs, fats, weight FROM food_catalog WHERE user_id = $1 ORDER BY name ASC", userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []CatalogFoodItem
	for rows.Next() {
		var item CatalogFoodItem
		if err := rows.Scan(&item.Name, &item.Calories, &item.Protein, &item.Carbs, &item.Fats, &item.Weight); err == nil {
			items = append(items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func CalculateWeightTrajectory(userID int) (currentWeight float64, weeklyTrend float64, daysRemaining float64, hasEnoughData bool, statusMsg string) {
	var goalWeight float64
	err := database.DB.QueryRow("SELECT goal_weight FROM user_stats WHERE user_id = $1", userID).Scan(&goalWeight)
	if err != nil || goalWeight <= 0 {
		return 0, 0, 0, false, "No goal weight set"
	}

	rows, err := database.DB.Query("SELECT log_date, weight FROM body_weight_logs WHERE user_id = $1 ORDER BY log_date DESC LIMIT 90", userID)
	if err != nil {
		return 0, 0, 0, false, "Not enough data"
	}
	defer rows.Close()

	var dates []time.Time
	var weights []float64

	for rows.Next() {
		var d string
		var w float64
		if err := rows.Scan(&d, &w); err == nil {
			if parsed, err := time.Parse("2006-01-02", d); err == nil {
				dates = append(dates, parsed)
				weights = append(weights, w)
			}
		}
	}

	if len(weights) < 7 {
		return 0, 0, 0, false, "Not enough data"
	}

	// Reverse to chronological order
	for i, j := 0, len(weights)-1; i < j; i, j = i+1, j-1 {
		weights[i], weights[j] = weights[j], weights[i]
		dates[i], dates[j] = dates[j], dates[i]
	}

	// Linear Regression
	// X = days since first log, Y = weight
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(weights))
	baseDate := dates[0]

	for i := 0; i < len(weights); i++ {
		x := dates[i].Sub(baseDate).Hours() / 24.0
		y := weights[i]

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope m (kg/day)
	denominator := (n * sumX2) - (sumX * sumX)
	var slope float64
	if denominator != 0 {
		slope = ((n * sumXY) - (sumX * sumY)) / denominator
	}

	weeklyTrend = slope * 7.0
	currentWeight = weights[len(weights)-1]

	if slope == 0 {
		return currentWeight, weeklyTrend, 0, true, "Maintaining weight perfectly."
	}

	if goalWeight < currentWeight {
		// Cutting phase
		if slope < 0 {
			daysRemaining = (currentWeight - goalWeight) / math.Abs(slope)
			statusMsg = fmt.Sprintf("↓ %.2f kg/week. Goal in ~%d days.", math.Abs(weeklyTrend), int(daysRemaining))
		} else {
			statusMsg = "You are gaining. Decrease calories to lose."
		}
	} else if goalWeight > currentWeight {
		// Gaining phase
		if slope > 0 {
			daysRemaining = (goalWeight - currentWeight) / math.Abs(slope)
			statusMsg = fmt.Sprintf("↑ %.2f kg/week. Goal in ~%d days.", math.Abs(weeklyTrend), int(daysRemaining))
		} else {
			statusMsg = "You are losing. Increase calories to gain."
		}
	} else {
		statusMsg = "You have reached your goal weight!"
	}

	return currentWeight, weeklyTrend, daysRemaining, true, statusMsg
}

func HandleMacroTargets(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	localDate, _ := getLocalTime(r)

	// Fetch Stats
	var age int
	var gender string
	var height, weight float64
	_ = database.DB.QueryRow("SELECT age, gender, height FROM user_stats WHERE user_id = $1", userID).Scan(&age, &gender, &height)
	_ = database.DB.QueryRow("SELECT weight FROM body_weight_logs WHERE user_id = $1 AND log_date <= $2 ORDER BY log_date DESC LIMIT 1", userID, localDate).Scan(&weight)

	if weight <= 0 {
		weight = 75
	}
	if height <= 0 {
		height = 175
	}
	if age <= 0 {
		age = 25
	}
	if gender == "" {
		gender = "male"
	}

	// Calculate BMR (Mifflin-St Jeor)
	bmr := 10*weight + 6.25*height - 5*float64(age)
	if gender == "male" {
		bmr += 5
	} else {
		bmr -= 161
	}

	// Active Calories Burnt (simplified average over last 7 days or just typical 1.2 multiplier)
	// For a static chart, we will use a standard activity multiplier of 1.375 (Lightly Active)
	// or 1.2 (Sedentary). Let's use 1.375 as they are using a fitness app.
	tdee := bmr * 1.375

	// Define speeds
	type Target struct {
		Label string
		Cal   int
		Pro   int
		Carb  int
		Fat   int
		Color string
	}

	calcMacros := func(cals float64) Target {
		// Minimum healthy calories
		if cals < 1200 {
			cals = 1200
		}

		pro := weight * 2.2 // 2.2g per kg
		fat := weight * 0.8 // 0.8g per kg

		proCals := pro * 4
		fatCals := fat * 9
		carbCals := cals - proCals - fatCals

		if carbCals < 0 {
			carbCals = 0
		}
		carb := carbCals / 4

		return Target{
			Cal:  int(math.Round(cals)),
			Pro:  int(math.Round(pro)),
			Carb: int(math.Round(carb)),
			Fat:  int(math.Round(fat)),
		}
	}

	targets := []Target{
		{Label: "Loss 2x", Cal: calcMacros(tdee - 1000).Cal, Pro: calcMacros(tdee - 1000).Pro, Carb: calcMacros(tdee - 1000).Carb, Fat: calcMacros(tdee - 1000).Fat, Color: "text-red-400"},
		{Label: "Loss 1x", Cal: calcMacros(tdee - 500).Cal, Pro: calcMacros(tdee - 500).Pro, Carb: calcMacros(tdee - 500).Carb, Fat: calcMacros(tdee - 500).Fat, Color: "text-orange-400"},
		{Label: "Loss 0.5x", Cal: calcMacros(tdee - 250).Cal, Pro: calcMacros(tdee - 250).Pro, Carb: calcMacros(tdee - 250).Carb, Fat: calcMacros(tdee - 250).Fat, Color: "text-yellow-400"},
		{Label: "Maintenance", Cal: calcMacros(tdee).Cal, Pro: calcMacros(tdee).Pro, Carb: calcMacros(tdee).Carb, Fat: calcMacros(tdee).Fat, Color: "text-zinc-300"},
		{Label: "Gain 0.5x", Cal: calcMacros(tdee + 250).Cal, Pro: calcMacros(tdee + 250).Pro, Carb: calcMacros(tdee + 250).Carb, Fat: calcMacros(tdee + 250).Fat, Color: "text-emerald-400"},
		{Label: "Gain 1x", Cal: calcMacros(tdee + 500).Cal, Pro: calcMacros(tdee + 500).Pro, Carb: calcMacros(tdee + 500).Carb, Fat: calcMacros(tdee + 500).Fat, Color: "text-blue-400"},
		{Label: "Gain 2x", Cal: calcMacros(tdee + 1000).Cal, Pro: calcMacros(tdee + 1000).Pro, Carb: calcMacros(tdee + 1000).Carb, Fat: calcMacros(tdee + 1000).Fat, Color: "text-indigo-400"},
	}

	html := `
	<div class="flex items-center justify-between mb-6 relative z-10 w-full">
		<h3 class="text-white font-black uppercase tracking-wider text-sm flex items-center gap-2">
			<svg class="w-5 h-5 text-app-pink" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
			Speed Targets
		</h3>
	</div>
	<div class="w-full relative z-10 overflow-x-auto custom-scrollbar">
		<table class="w-full text-left border-collapse min-w-[300px]">
			<thead>
				<tr class="border-b border-white/5">
					<th class="py-2 text-[10px] uppercase font-bold text-zinc-500 tracking-wider">Goal</th>
					<th class="py-2 text-[10px] uppercase font-bold text-zinc-500 tracking-wider text-right">KCAL</th>
					<th class="py-2 text-[10px] uppercase font-bold text-zinc-500 tracking-wider text-right">PRO</th>
					<th class="py-2 text-[10px] uppercase font-bold text-zinc-500 tracking-wider text-right">CRB</th>
					<th class="py-2 text-[10px] uppercase font-bold text-zinc-500 tracking-wider text-right">FAT</th>
				</tr>
			</thead>
			<tbody>`

	for _, t := range targets {
		html += fmt.Sprintf(`
				<tr class="border-b border-white/5 hover:bg-white/5 transition-colors">
					<td class="py-3 text-xs font-black uppercase tracking-tight %s">%s</td>
					<td class="py-3 text-sm font-mono font-bold text-white text-right">%d</td>
					<td class="py-3 text-xs font-mono font-bold text-zinc-400 text-right">%d</td>
					<td class="py-3 text-xs font-mono font-bold text-zinc-400 text-right">%d</td>
					<td class="py-3 text-xs font-mono font-bold text-zinc-400 text-right">%d</td>
				</tr>`, t.Color, t.Label, t.Cal, t.Pro, t.Carb, t.Fat)
	}

	html += `
			</tbody>
		</table>
	</div>`

	w.Write([]byte(html))
}
