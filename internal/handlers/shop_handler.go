package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gama-fit/database"
)

type shopEntry struct {
	ID       int
	Category string
	Name     string
	Cost     int
	Owned    int
}

func renderShop(w http.ResponseWriter, userID int, message string) {
	var totalCoins int
	_ = database.DB.QueryRow("SELECT total_coins FROM user_stats WHERE user_id = $1", userID).Scan(&totalCoins)

	rows, err := database.DB.Query("SELECT id, category, name, cost, owned FROM shop_catalog WHERE user_id = $1 ORDER BY category ASC, id ASC", userID)
	if err != nil {
		fmt.Fprint(w, `<div id="shop-container" class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden animate-fade-in-up">
			<div class="text-zinc-500 text-sm text-center py-8">Shop unavailable.</div>
		</div>`)
		return
	}
	defer rows.Close()

	var activities, items []shopEntry
	for rows.Next() {
		var e shopEntry
		if err := rows.Scan(&e.ID, &e.Category, &e.Name, &e.Cost, &e.Owned); err == nil {
			if e.Category == "activity" {
				activities = append(activities, e)
			} else {
				items = append(items, e)
			}
		}
	}

	buildCard := func(title, accent, category string, entries []shopEntry, placeholder string) string {
		html := fmt.Sprintf(`
			<div class="bg-zinc-900/30 border border-white/5 rounded-[1.75rem] p-5 lg:p-6 flex flex-col">
				<div class="flex items-start justify-between gap-4 mb-5">
					<div>
						<h4 class="text-white font-black uppercase tracking-wider text-sm">%s</h4>
						<p class="text-zinc-500 text-xs mt-1">%s</p>
					</div>
					<div class="px-3 py-1 rounded-full bg-%s/10 text-%s text-[10px] font-bold uppercase tracking-widest border border-%s/20">%s</div>
				</div>
				<form hx-post="/api/shop?category=%s" hx-target="#shop-container" hx-swap="outerHTML" hx-on::after-request="this.reset()" class="flex flex-col sm:flex-row gap-3 mb-5">
					<input type="text" name="name" placeholder="%s" required class="flex-1 bg-zinc-900/60 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-pink transition-colors">
					<input type="number" name="cost" min="1" value="25" required class="w-full sm:w-24 bg-zinc-900/60 border border-zinc-700 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-app-yellow transition-colors">
					<button type="submit" class="bg-app-pink text-white font-bold px-5 rounded-xl hover:bg-pink-500 transition-all shadow-[0_0_15px_rgba(255,0,160,0.25)] flex items-center justify-center">+</button>
				</form>
				<div class="space-y-2 flex-1 max-h-72 overflow-y-auto pr-1 scrollbar-hide">`, title, strings.TrimSpace(placeholder), accent, accent, accent, strings.Title(category), category, placeholder)
		if len(entries) == 0 {
			html += `<div class="text-zinc-600 text-center py-6 font-mono text-sm">Nothing in the shop yet.</div>`
		}
		for _, e := range entries {
			canBuy := totalCoins >= e.Cost
			buyBtn := ""
			if canBuy {
				buyBtn = fmt.Sprintf(`<button hx-post="/api/shop/buy?id=%d" hx-target="#shop-container" hx-swap="outerHTML" class="bg-app-yellow text-black font-black px-4 py-2 rounded-lg hover:bg-yellow-400 transition-all text-[10px] uppercase tracking-wider">Buy</button>`, e.ID)
			} else {
				buyBtn = fmt.Sprintf(`<button disabled class="bg-zinc-800 text-zinc-500 font-black px-4 py-2 rounded-lg text-[10px] uppercase tracking-wider cursor-not-allowed">Need %d</button>`, e.Cost)
			}
			html += fmt.Sprintf(`
				<div class="flex items-center justify-between bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 group hover:border-zinc-600 transition-colors">
					<div class="min-w-0 pr-3">
						<div class="text-white text-sm font-bold truncate">%s</div>
						<div class="text-zinc-500 text-[10px] uppercase font-bold tracking-widest">%d GC • Owned x%d</div>
					</div>
					<div class="flex items-center gap-2 shrink-0">
						%s
						<button hx-delete="/api/shop?id=%d" hx-target="#shop-container" hx-swap="outerHTML" class="text-zinc-600 hover:text-red-500 transition-colors p-1">
							<svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
						</button>
					</div>
				</div>`, e.Name, e.Cost, e.Owned, buyBtn, e.ID)
		}
		html += `</div></div>`
		return html
	}

	var panel strings.Builder
	panel.WriteString(`<div id="shop-container" class="glass-panel rounded-[2rem] p-6 lg:p-8 relative overflow-hidden animate-fade-in-up mt-6 lg:mt-8">`)
	panel.WriteString(`<div class="absolute top-0 right-0 w-64 h-64 bg-app-yellow/10 blur-3xl rounded-full translate-x-1/3 -translate-y-1/3 pointer-events-none"></div>`)
	panel.WriteString(`<div class="flex flex-col lg:flex-row lg:items-end justify-between gap-4 mb-6 relative z-10">`)
	panel.WriteString(`<div><h3 class="text-white font-black uppercase tracking-wider text-sm">Activities & Items</h3><p class="text-zinc-500 text-xs mt-1">Spend coins, collect rewards, and build your own shop.</p></div>`)
	panel.WriteString(fmt.Sprintf(`<div class="inline-flex items-center gap-2 bg-zinc-900/70 border border-zinc-800 rounded-full px-4 py-2 text-[10px] font-bold uppercase tracking-widest text-app-yellow">Balance %d GC</div>`, totalCoins))
	panel.WriteString(`</div>`)
	if strings.TrimSpace(message) != "" {
		panel.WriteString(fmt.Sprintf(`<div class="mb-6 rounded-xl border border-app-yellow/20 bg-app-yellow/10 px-4 py-3 text-sm text-app-yellow">%s</div>`, message))
	}
	panel.WriteString(`<div class="grid grid-cols-1 xl:grid-cols-2 gap-6 relative z-10">`)
	panel.WriteString(buildCard("Activities", "app-pink", "activity", activities, "Add new activity..."))
	panel.WriteString(buildCard("Items", "app-yellow", "item", items, "Add new item..."))
	panel.WriteString(`</div></div>`)

	fmt.Fprint(w, panel.String())
}

func HandleShop(w http.ResponseWriter, r *http.Request) {
	userID, _ := GetUserID(r)
	category := r.URL.Query().Get("category")
	action := r.URL.Query().Get("action")
	id := r.URL.Query().Get("id")

	path := strings.TrimPrefix(r.URL.Path, "/api/shop")
	path = strings.TrimPrefix(path, "/")

	// Support both:
	//   /api/shop?action=buy&id=...
	//   /api/shop/buy?id=...
	if action == "buy" || path == "buy" {
		if r.Method != http.MethodPost {
			renderShop(w, userID, "")
			return
		}
		handleShopBuy(w, userID, id)
		return
	}

	switch r.Method {
	case http.MethodPost:
		_ = r.ParseForm()
		handleShopCreate(w, userID, category, r.FormValue("name"), r.FormValue("cost"))
		return
	case http.MethodDelete:
		if id != "" {
			_, _ = database.DB.Exec("DELETE FROM shop_catalog WHERE id = $1 AND user_id = $2", id, userID)
		}
		renderShop(w, userID, "")
		return
	default:
		renderShop(w, userID, "")
		return
	}
}

func handleShopCreate(w http.ResponseWriter, userID int, category, name, costStr string) {
	cost, err := strconv.Atoi(costStr)
	if err != nil || strings.TrimSpace(name) == "" {
		renderShop(w, userID, "Invalid shop entry.")
		return
	}

	if category != "activity" && category != "item" {
		category = "item"
	}

	_, err = database.DB.Exec(
		`INSERT INTO shop_catalog (user_id, category, name, cost, owned) VALUES ($1, $2, $3, $4, 0)`,
		userID, category, strings.TrimSpace(name), cost,
	)
	if err != nil {
		renderShop(w, userID, "Could not add shop item.")
		return
	}

	renderShop(w, userID, "Shop updated.")
}

func handleShopBuy(w http.ResponseWriter, userID int, id string) {
	if strings.TrimSpace(id) == "" {
		renderShop(w, userID, "Invalid item.")
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		renderShop(w, userID, "Transaction failed.")
		return
	}
	defer tx.Rollback()

	var cost int
	var owned int
	err = tx.QueryRow(`SELECT cost, owned FROM shop_catalog WHERE id = $1 AND user_id = $2`, id, userID).Scan(&cost, &owned)
	if err != nil {
		renderShop(w, userID, "Item not found.")
		return
	}

	var coins int
	err = tx.QueryRow(`SELECT total_coins FROM user_stats WHERE user_id = $1`, userID).Scan(&coins)
	if err != nil {
		renderShop(w, userID, "Coin account not found.")
		return
	}

	if coins < cost {
		renderShop(w, userID, "Not enough coins.")
		return
	}

	_, err = tx.Exec(`UPDATE user_stats SET total_coins = total_coins - $1 WHERE user_id = $2`, cost, userID)
	if err != nil {
		renderShop(w, userID, "Failed to deduct coins.")
		return
	}

	_, err = tx.Exec(`UPDATE shop_catalog SET owned = owned + 1 WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		renderShop(w, userID, "Failed to update ownership.")
		return
	}

	if err = tx.Commit(); err != nil {
		renderShop(w, userID, "Purchase could not be completed.")
		return
	}

	renderShop(w, userID, "Purchased.")
}
