package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"gama-fit/database"
)

type ChartPoint struct {
	Label  string
	Weight float64
	Reps   int
}

func HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	timeframe := normalizeTimeframe(r.URL.Query().Get("timeframe"))
	selectedExercise := strings.TrimSpace(r.URL.Query().Get("exercise"))
	if selectedExercise == "" {
		selectedExercise = "all"
	}

	now := time.Now()

	avgSleepHours := fetchAverageSleepHours()
	avgCalories, avgProtein := fetchAverageNutrition()

	statsHTML := fmt.Sprintf(`
		<div class="grid grid-cols-1 md:grid-cols-3 gap-5 mb-6">
			%s
			%s
			%s
		</div>
	`,
		statCard("Avg Sleep", fmt.Sprintf("%.1f", avgSleepHours), "hours", "rgba(99,102,241,0.12)", "text-app-sleep"),
		statCard("Avg Calories", fmt.Sprintf("%d", avgCalories), "kcal", "rgba(251,255,0,0.10)", "text-app-yellow"),
		statCard("Avg Protein", fmt.Sprintf("%d", avgProtein), "g", "rgba(255,0,160,0.10)", "text-app-pink"),
	)

	fragment := fmt.Sprintf(`
		<div class="space-y-6">
			<div class="flex flex-col lg:flex-row lg:items-end justify-between gap-4">
				<div>
					<p class="text-fuchsia-500 text-[10px] font-black uppercase tracking-[0.35em] mb-2">Analyze the machine</p>
					<h2 class="text-3xl md:text-4xl font-black text-white tracking-tight">Analytics</h2>
					<p class="text-zinc-500 text-sm mt-2">Graphical progression tracking over time.</p>
				</div>
			</div>

			%s

			<div class="glass-panel rounded-[2.5rem] p-5 lg:p-7 relative overflow-hidden border border-zinc-800/80 shadow-2xl">
				<div class="absolute top-0 right-0 w-96 h-96 rounded-full blur-[150px] pointer-events-none" style="background:rgba(255,0,160,0.05);"></div>

				<div class="flex flex-col lg:flex-row lg:items-end justify-between gap-4 mb-8 relative z-10">
					<div>
						<h3 class="text-white font-black uppercase tracking-wider text-sm flex items-center gap-2 mb-1">
							<span style="color:#ff00a0">⚡</span> Progression Graph
						</h3>
						<p class="text-zinc-500 text-xs font-bold uppercase tracking-widest">Weight (kg) vs Time</p>
					</div>

					<div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 w-full lg:w-auto">
						<select id="exercise-select" class="bg-zinc-900 border border-zinc-800 hover:border-app-pink/50 text-white px-4 py-3 rounded-xl text-xs font-bold uppercase tracking-wider transition-all outline-none cursor-pointer shadow-lg w-full lg:w-72 appearance-none">
							%s
						</select>

						<div class="grid grid-cols-3 sm:flex gap-2">
							%s
							%s
							%s
						</div>
					</div>
				</div>

				<div class="relative z-10 w-full">
					%s
				</div>
			</div>
		</div>
	`,
		statsHTML,
		buildExerciseOptions(selectedExercise),
		timeframeButton("week", timeframe),
		timeframeButton("month", timeframe),
		timeframeButton("year", timeframe),
		renderAreaChart(fetchGraphPoints(timeframe, selectedExercise, now)),
	)

	fmt.Fprint(w, fragment)
}

func normalizeTimeframe(value string) string {
	switch value {
	case "week", "month", "year":
		return value
	default:
		return "week"
	}
}

func fetchGraphPoints(timeframe, exercise string, now time.Time) []ChartPoint {
	var points []ChartPoint
	if timeframe == "week" {
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			where := "logged_date = ?"
			args := []any{day.Format("2006-01-02")}
			if exercise != "all" {
				where += " AND exercise_name = ?"
				args = append(args, exercise)
			}
			weight, reps := queryMaxSet(where, args...)
			if weight > 0 && !math.IsNaN(weight) && !math.IsInf(weight, 0) {
				points = append(points, ChartPoint{Label: day.Format("Mon"), Weight: weight, Reps: reps})
			}
		}
	} else if timeframe == "month" {
		for i := 3; i >= 0; i-- {
			start := now.AddDate(0, 0, -((i+1)*7 - 1))
			end := start.AddDate(0, 0, 6)
			where := "logged_date >= ? AND logged_date <= ?"
			args := []any{start.Format("2006-01-02"), end.Format("2006-01-02")}
			if exercise != "all" {
				where += " AND exercise_name = ?"
				args = append(args, exercise)
			}
			weight, reps := queryMaxSet(where, args...)
			if weight > 0 && !math.IsNaN(weight) && !math.IsInf(weight, 0) {
				points = append(points, ChartPoint{Label: fmt.Sprintf("W%d", 4-i), Weight: weight, Reps: reps})
			}
		}
	} else if timeframe == "year" {
		for i := 11; i >= 0; i-- {
			month := now.AddDate(0, -i, 0)
			where := "logged_date LIKE ?"
			args := []any{month.Format("2006-01") + "%"}
			if exercise != "all" {
				where += " AND exercise_name = ?"
				args = append(args, exercise)
			}
			weight, reps := queryMaxSet(where, args...)
			if weight > 0 && !math.IsNaN(weight) && !math.IsInf(weight, 0) {
				points = append(points, ChartPoint{Label: month.Format("Jan"), Weight: weight, Reps: reps})
			}
		}
	}
	return points
}

func queryMaxSet(where string, args ...any) (float64, int) {
	query := "SELECT weight, reps FROM freestyle_logs WHERE " + where + " ORDER BY weight DESC LIMIT 1"
	var w sql.NullFloat64
	var r sql.NullInt64
	err := database.DB.QueryRow(query, args...).Scan(&w, &r)
	if err != nil || !w.Valid || !r.Valid {
		return 0, 0
	}
	return w.Float64, int(r.Int64)
}

func renderAreaChart(points []ChartPoint) string {
	var validPoints []ChartPoint
	for _, p := range points {
		if !math.IsNaN(p.Weight) && !math.IsInf(p.Weight, 0) {
			validPoints = append(validPoints, p)
		}
	}
	if len(validPoints) < 2 {
		return `<div class="h-[400px] flex items-center justify-center text-zinc-500 bg-zinc-900/20 rounded-3xl border border-zinc-800/50">
					<span class="text-sm font-bold uppercase tracking-widest text-zinc-600">Need at least 2 logs.</span>
				</div>`
	}

	minW, maxW := validPoints[0].Weight, validPoints[0].Weight
	for _, p := range validPoints {
		if p.Weight < minW { minW = p.Weight }
		if p.Weight > maxW { maxW = p.Weight }
	}
	rangeY := (maxW + (maxW-minW)*0.25) - (minW - (maxW-minW)*0.25)
	if rangeY <= 0 { rangeY = 1 }
	bottomY := minW - (maxW-minW)*0.25

	// 1. ADD PADDING: Keep dots inside the safe zone (5% to 95%)
	type Coord struct { X, Y float64 }
	coords := make([]Coord, len(validPoints))
	paddingX, rangeX := 5.0, 90.0 
	divider := float64(len(validPoints) - 1)
	if divider <= 0 { divider = 1 }

	for i, p := range validPoints {
		progress := float64(i) / divider
		x := paddingX + (progress * rangeX)
		y := 100 - (((p.Weight - bottomY) / rangeY) * 100)
		coords[i] = Coord{X: x, Y: y}
	}

	var pathLine, pathArea strings.Builder
	
	// 2. EDGE-TO-EDGE LINE: Start at absolute left wall, draw to first dot
	pathArea.WriteString(fmt.Sprintf("M 0,100 L 0,%.2f L %.2f,%.2f ", coords[0].Y, coords[0].X, coords[0].Y))
	pathLine.WriteString(fmt.Sprintf("M 0,%.2f L %.2f,%.2f ", coords[0].Y, coords[0].X, coords[0].Y))

	for i := 0; i < len(coords)-1; i++ {
		prev, curr := coords[i], coords[i+1]
		cpX1, cpY1 := prev.X+(curr.X-prev.X)*0.5, prev.Y
		cpX2, cpY2 := curr.X-(curr.X-prev.X)*0.5, curr.Y
		curve := fmt.Sprintf("C %.2f,%.2f %.2f,%.2f %.2f,%.2f ", cpX1, cpY1, cpX2, cpY2, curr.X, curr.Y)
		pathLine.WriteString(curve)
		pathArea.WriteString(curve)
	}
	
	// 3. EDGE-TO-EDGE LINE: Draw from last dot to absolute right wall
	pathLine.WriteString(fmt.Sprintf("L 100,%.2f ", coords[len(coords)-1].Y))
	pathArea.WriteString(fmt.Sprintf("L 100,%.2f L 100,100 Z", coords[len(coords)-1].Y))

	var b strings.Builder
	b.WriteString(`
	<style>
		@keyframes drawLine { to { stroke-dashoffset: 0; } }
		.neon-path { stroke-dasharray: 2000; stroke-dashoffset: 2000; animation: drawLine 1.5s cubic-bezier(0.16, 1, 0.3, 1) forwards; }
	</style>
	<div class="w-full mt-6">
		<div class="relative w-full h-[220px] sm:h-[300px]">
			<svg class="absolute inset-0 w-full h-full overflow-visible" viewBox="0 0 100 100" preserveAspectRatio="none">
				<path d="` + pathArea.String() + `" fill="#ff00a0" fill-opacity="0.1" />
				<path d="` + pathLine.String() + `" fill="none" stroke="#ff00a0" stroke-width="3px" vector-effect="non-scaling-stroke" class="neon-path" />
			</svg>
	`)

	for i, p := range validPoints {
		c := coords[i]
		
		// Prevent Tooltip from clipping edges
		ttPos := "left-1/2 -translate-x-1/2"
		if i == 0 { ttPos = "left-0 -translate-x-[10%%]" }
		if i == len(validPoints)-1 { ttPos = "right-0 translate-x-[10%%]" }

		b.WriteString(fmt.Sprintf(`
			<div class="absolute z-30 group" style="left: %.2f%%; top: %.2f%%; transform: translate(-50%%, -50%%);">
				
				<div class="w-12 h-12 flex items-center justify-center">
					<div class="w-3 h-3 md:w-4 md:h-4 rounded-full bg-app-pink shadow-[0_0_10px_#ff00a0] group-hover:scale-[1.8] transition-transform"></div>
				</div>
				
				<div class="absolute bottom-full mb-1 bg-zinc-900 border border-app-pink/50 p-3 rounded-xl shadow-2xl opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity min-w-max text-center %s">
					<div class="text-white font-black text-xl">%s <span class="text-xs text-app-pink">KG</span></div>
					<div class="text-zinc-400 font-bold text-xs uppercase tracking-widest mt-1">× %d Reps</div>
					<div class="text-zinc-600 font-bold text-[10px] uppercase mt-2">%s</div>
				</div>

			</div>
		`, c.X, c.Y, ttPos, formatKg(p.Weight), p.Reps, p.Label))
	}

	b.WriteString(`</div>`)
	
	// Dynamic Labels locked directly under the dots
	b.WriteString(`<div class="relative w-full h-10 mt-2 border-t border-zinc-800/80 pt-3">`)
	for i, p := range validPoints {
		b.WriteString(fmt.Sprintf(`<div class="absolute top-3 text-zinc-500 font-black text-[10px] uppercase -translate-x-1/2" style="left: %.2f%%;">%s</div>`, coords[i].X, p.Label))
	}
	b.WriteString(`</div></div>`)
	
	return b.String()
}
func timeframeButton(value, active string) string {
	base := "px-4 py-3 rounded-xl text-xs font-bold uppercase tracking-wider transition-all duration-300 whitespace-nowrap text-center flex-1 sm:flex-none"
	if value == active {
		return fmt.Sprintf(`<button type="button" data-timeframe="%s" class="%s bg-app-pink text-white shadow-[0_0_15px_rgba(255,0,160,0.35)]">%s</button>`, value, base, strings.Title(value))
	}
	return fmt.Sprintf(`<button type="button" data-timeframe="%s" class="%s bg-zinc-900 text-zinc-400 border border-zinc-800 hover:border-zinc-700 hover:text-white">%s</button>`, value, base, strings.Title(value))
}

func fetchAverageSleepHours() float64 {
	var avg float64
	_ = database.DB.QueryRow(`SELECT COALESCE(AVG(duration_mins), 0) / 60.0 FROM sleep_logs WHERE log_date >= date('now', '-6 days')`).Scan(&avg)
	return avg
}

func fetchAverageNutrition() (int, int) {
	var totalCalories, totalProtein int
	_ = database.DB.QueryRow(`SELECT COALESCE(SUM(calories), 0), COALESCE(SUM(protein), 0) FROM daily_meals WHERE log_date >= date('now', '-6 days')`).Scan(&totalCalories, &totalProtein)
	return totalCalories / 7, totalProtein / 7
}

func buildExerciseOptions(selected string) string {
	rows, err := database.DB.Query(`SELECT DISTINCT exercise_name FROM freestyle_logs WHERE exercise_name IS NOT NULL AND exercise_name != '' ORDER BY exercise_name ASC`)
	if err != nil { return `<option value="all" selected>All exercises</option>` }
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && strings.TrimSpace(name) != "" { names = append(names, name) }
	}
	if len(names) == 0 { return `<option value="all" selected>All exercises</option>` }
	var b strings.Builder
	b.WriteString(`<option value="all"`)
	if selected == "all" { b.WriteString(` selected`) }
	b.WriteString(`>All exercises</option>`)
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(`<option value="`)
		b.WriteString(htmlEscape(name))
		b.WriteString(`"`)
		if name == selected { b.WriteString(` selected`) }
		b.WriteString(`>`)
		b.WriteString(htmlEscape(name))
		b.WriteString(`</option>`)
	}
	return b.String()
}

func statCard(title, value, unit, glow, accentClass string) string {
	return fmt.Sprintf(`
		<div class="glass-panel rounded-[2rem] p-6 relative overflow-hidden">
			<div class="absolute top-0 right-0 w-24 h-24 rounded-full blur-2xl pointer-events-none" style="background:%s;"></div>
			<div class="absolute top-5 right-5 text-xs font-black uppercase tracking-widest %s">●</div>
			<p class="text-zinc-500 text-xs font-bold uppercase tracking-widest relative z-10">%s</p>
			<div class="mt-3 flex items-end gap-2 relative z-10">
				<h3 class="text-white font-black text-4xl leading-none">%s</h3>
				<span class="text-zinc-400 text-sm font-bold pb-0.5">%s</span>
			</div>
		</div>
	`, glow, accentClass, title, value, unit)
}

func formatKg(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) { return "0" }
	if math.Abs(v-math.Round(v)) < 0.05 { return fmt.Sprintf("%.0f", math.Round(v)) }
	return fmt.Sprintf("%.1f", v)
}

func htmlEscape(s string) string {
	return strings.NewReplacer(`&`, `&amp;`, `<`, `&lt;`, `>`, `&gt;`, `"`, `&quot;`, `'`, `&#39;`).Replace(s)
}