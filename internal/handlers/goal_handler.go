package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"gama-fit/database"
	"gama-fit/models"
)

type TemplateData struct {
	Goals      []models.Goal
	TotalCoins int
}

const goalsHTML = `
{{range .Goals}}
<div id="goal-{{.ID}}" class="flex items-center justify-between p-3 rounded-xl border transition-all {{if .Completed}}bg-zinc-900/80 border-app-yellow/30{{else}}bg-zinc-900/40 border-white/5 hover:bg-zinc-900/80{{end}}">
	<div class="flex items-center gap-3">
		{{if .Completed}}
			<div class="w-5 h-5 rounded flex items-center justify-center shrink-0 text-app-yellow">
				<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
			</div>
		{{else}}
			<div hx-put="/api/goals/{{.ID}}/toggle" hx-target="#goals-container" class="w-5 h-5 rounded border-2 border-zinc-600 flex items-center justify-center shrink-0 hover:border-app-yellow transition-colors cursor-pointer"></div>
		{{end}}
		<span class="text-sm {{if .Completed}}text-zinc-500 line-through{{else}}text-zinc-300 font-medium{{end}}">{{.Title}}</span>
	</div>
	
	<div class="flex items-center gap-2">
		{{if not .Completed}}
			<span class="text-[10px] font-bold uppercase tracking-wider text-zinc-500 bg-zinc-800 px-2 py-1 rounded">{{.Reward}} GC</span>
		{{else if and .Completed (not .Claimed)}}
			<button hx-put="/api/goals/{{.ID}}/claim" hx-target="#goals-container" class="text-[10px] font-bold uppercase tracking-wider bg-app-pink text-white px-3 py-1.5 rounded-lg hover:bg-app-yellow hover:text-black transition-all">Claim {{.Reward}}</button>
		{{else}}
			<span class="text-[10px] font-bold uppercase tracking-wider text-zinc-500">Claimed</span>
		{{end}}
		
		<button hx-delete="/api/goals/{{.ID}}" hx-target="#goals-container" class="p-1.5 text-zinc-600 hover:text-red-500 hover:bg-red-500/10 rounded transition-colors" title="Remove Goal">
			<svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
		</button>
	</div>
</div>
{{end}}

<div id="coin-widget" hx-swap-oob="true" class="flex items-center gap-2 bg-app-card border border-app-pink/20 rounded-xl px-5 py-3 w-full sm:w-auto justify-center sm:justify-start shadow-[0_0_15px_rgba(255,0,160,0.1)]">
	<svg class="w-5 h-5 text-app-yellow" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10 10-4.477 10-10 10zm0-2a8 8 0 1 0 0-16 8 8 0 0 0 0 16zm-2-7.5l6-4.5v9l-6-4.5z"/></svg>
	<span id="total-coins" class="font-display font-black text-white text-lg transition-all duration-300">{{.TotalCoins}}</span>
	<span class="text-zinc-500 font-sans font-bold tracking-wider">GC</span>
</div>
`

var tpl = template.Must(template.New("goals").Parse(goalsHTML))

func renderGoals(w http.ResponseWriter) {
	rows, err := database.DB.Query("SELECT id, title, completed, claimed, reward FROM goals ORDER BY id ASC")
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var data TemplateData
	for rows.Next() {
		var g models.Goal
		var completed, claimed int
		if err := rows.Scan(&g.ID, &g.Title, &completed, &claimed, &g.Reward); err == nil {
			g.Completed = completed == 1
			g.Claimed = claimed == 1
			data.Goals = append(data.Goals, g)
		}
	}

	_ = database.DB.QueryRow("SELECT total_coins FROM user_stats WHERE id = 1").Scan(&data.TotalCoins)
	_ = tpl.Execute(w, data)
}

func HandleGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		rewardStr := r.FormValue("reward")
		reward := 50
		if rewardStr != "" {
			if rv, err := strconv.Atoi(rewardStr); err == nil {
				reward = rv
			}
		}
		if title != "" {
			_, err := database.DB.Exec("INSERT INTO goals (title, reward) VALUES (?, ?)", title, reward)
			if err != nil {
				fmt.Printf("DATABASE ERROR: %v\n", err)
			} else {
				fmt.Printf("GOAL ADDED: %s with reward %d\n", title, reward)
			}
		}
	}
	renderGoals(w)
}

func HandleGoalActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/goals/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		renderGoals(w)
		return
	}
	id := parts[0]

	if len(parts) == 2 && parts[1] == "toggle" {
		_, _ = database.DB.Exec("UPDATE goals SET completed = CASE WHEN completed = 1 THEN 0 ELSE 1 END WHERE id = ?", id)
	} else if len(parts) == 2 && parts[1] == "claim" {
		var reward int
		err := database.DB.QueryRow("SELECT reward FROM goals WHERE id = ? AND completed = 1 AND claimed = 0", id).Scan(&reward)
		if err == nil {
			_, _ = database.DB.Exec("UPDATE goals SET claimed = 1, claimed_at = date('now') WHERE id = ?", id)
			_, _ = database.DB.Exec("UPDATE user_stats SET total_coins = total_coins + ? WHERE id = 1", reward)
		}
	} else if r.Method == "DELETE" {
		_, _ = database.DB.Exec("DELETE FROM goals WHERE id = ?", id)
	}

	renderGoals(w)
}
