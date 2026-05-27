package handlers

import (
	"fmt"
	"net/http"

	"gama-fit/database"
)

func GetCoins(w http.ResponseWriter, r *http.Request) {
	var totalCoins int
	_ = database.DB.QueryRow("SELECT total_coins FROM user_stats WHERE id = 1").Scan(&totalCoins)

	html := `
	<div id="coin-widget" class="flex items-center gap-2 bg-app-card border border-app-pink/20 rounded-xl px-5 py-3 w-full sm:w-auto justify-center sm:justify-start shadow-[0_0_15px_rgba(255,0,160,0.1)]">
		<svg class="w-5 h-5 text-app-yellow" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10 10-4.477 10-10 10zm0-2a8 8 0 1 0 0-16 8 8 0 0 0 0 16zm-2-7.5l6-4.5v9l-6-4.5z"/></svg>
		<span id="total-coins" class="font-display font-black text-white text-lg transition-all duration-300">%d</span>
		<span class="text-zinc-500 font-sans font-bold tracking-wider">GC</span>
	</div>`

	fmt.Fprintf(w, html, totalCoins)
}
