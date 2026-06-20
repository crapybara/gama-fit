package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8095"
	cookieFile     = ".gama_session"
	bedtimeFile    = ".gama_bedtime"
)

var dynamicBaseURL string

func init() {
	dynamicBaseURL = os.Getenv("GAMA_URL")
	if dynamicBaseURL == "" {
		dynamicBaseURL = defaultBaseURL
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

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// support both "-food" and "food" for backwards compatibility
	command := strings.TrimPrefix(os.Args[1], "-")
	args := os.Args[2:]

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	loadSession(jar)

	switch command {
	case "login":
		if len(args) < 2 {
			fmt.Println("Usage: gama login <username> <password>")
			return
		}
		login(client, args[0], args[1])

	case "foodslist":
		listFoods(client)

	case "food":
		if len(args) == 0 {
			fmt.Println("Usage:")
			fmt.Println("  gama food <number from catalog>")
			fmt.Println("  gama food \"Name from catalog\"")
			fmt.Println("  gama food \"Custom Food\" -k 250 -p 33 -c 55 -f 20")
			return
		}

		// Check if it's a number
		if idx, err := strconv.Atoi(args[0]); err == nil && len(args) == 1 {
			logFoodByNumber(client, idx)
			return
		}

		foodName := args[0]
		var k, p, c, f string
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-k":
				if i+1 < len(args) { k = args[i+1]; i++ }
			case "-p":
				if i+1 < len(args) { p = args[i+1]; i++ }
			case "-c":
				if i+1 < len(args) { c = args[i+1]; i++ }
			case "-f":
				if i+1 < len(args) { f = args[i+1]; i++ }
			}
		}

		if k != "" || p != "" || c != "" || f != "" {
			logQuickFood(client, foodName, k, p, c, f)
		} else {
			logFoodByName(client, foodName)
		}

	case "sleep":
		timeStr := time.Now().Format("15:04")
		if len(args) > 0 {
			timeStr = args[0]
		}
		home, _ := os.UserHomeDir()
		os.WriteFile(filepath.Join(home, bedtimeFile), []byte(timeStr), 0600)
		fmt.Printf("Bedtime recorded at: %s\nRun 'gama woke' in the morning!\n", timeStr)

	case "woke":
		timeStr := time.Now().Format("15:04")
		if len(args) > 0 {
			timeStr = args[0]
		}
		home, _ := os.UserHomeDir()
		b, err := os.ReadFile(filepath.Join(home, bedtimeFile))
		if err != nil {
			fmt.Println("No bedtime recorded! Run 'gama sleep' first or provide both times directly.")
			return
		}
		bedtime := string(b)
		logSleep(client, bedtime, timeStr, "avg")
		os.Remove(filepath.Join(home, bedtimeFile))

	case "checkin":
		dateStr := time.Now().Format("2006-01-02")
		if len(args) > 0 {
			dateStr = args[0]
		}
		logCheckin(client, dateStr)

	case "weight":
		if len(args) == 0 {
			fmt.Println("Usage: gama weight <kg>")
			return
		}
		logWeight(client, args[0])

	default:
		printUsage()
	}
}

func login(client *http.Client, username, password string) {
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)

	resp, err := client.PostForm(dynamicBaseURL+"/api/login", data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Login successful!")
		saveSession(client.Jar)
	} else {
		fmt.Printf("Login failed (Status: %d)\n", resp.StatusCode)
	}
}

func listFoods(client *http.Client) {
	resp, err := client.Get(dynamicBaseURL + "/api/foods/catalog")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Not logged in or catalog unavailable.")
		return
	}

	var items []CatalogFoodItem
	json.NewDecoder(resp.Body).Decode(&items)
	if len(items) == 0 {
		fmt.Println("Your food catalog is empty.")
		return
	}
	fmt.Println("Food Catalog:")
	for i, item := range items {
		fmt.Printf("%d. %s (%.0fg) - %d kcal | %.1fP %.1fC %.1fF\n", i+1, item.Name, item.Weight, item.Calories, item.Protein, item.Carbs, item.Fats)
	}
}

func logFoodByNumber(client *http.Client, pos int) {
	items := fetchCatalog(client)
	if pos < 1 || pos > len(items) {
		fmt.Println("Invalid food item number.")
		return
	}
	logCatalogFood(client, items[pos-1])
}

func logFoodByName(client *http.Client, name string) {
	items := fetchCatalog(client)
	nameLower := strings.ToLower(name)
	for _, item := range items {
		if strings.ToLower(item.Name) == nameLower {
			logCatalogFood(client, item)
			return
		}
	}
	fmt.Printf("Could not find '%s' in your catalog. Use -k -p -c -f to log it as a quick food.\n", name)
}

func fetchCatalog(client *http.Client) []CatalogFoodItem {
	resp, err := client.Get(dynamicBaseURL + "/api/foods/catalog")
	if err != nil { return nil }
	defer resp.Body.Close()
	var items []CatalogFoodItem
	json.NewDecoder(resp.Body).Decode(&items)
	return items
}

func logCatalogFood(client *http.Client, item CatalogFoodItem) {
	data := url.Values{}
	data.Set("catalog_food", item.Name)
	data.Set("catalog_grams", fmt.Sprintf("%.0f", item.Weight))
	postResp, err := client.PostForm(dynamicBaseURL+"/api/meals", data)
	if err == nil && postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully logged %s (%d kcal)!\n", item.Name, item.Calories)
	} else {
		fmt.Println("Failed to log meal.")
	}
	if postResp != nil { postResp.Body.Close() }
}

func logQuickFood(client *http.Client, name, k, p, c, f string) {
	data := url.Values{}
	data.Set("quick_name", name)
	if k != "" { data.Set("quick_calories", k) }
	if p != "" { data.Set("quick_protein", p) }
	if c != "" { data.Set("quick_carbs", c) }
	if f != "" { data.Set("quick_fats", f) }
	postResp, err := client.PostForm(dynamicBaseURL+"/api/meals", data)
	if err == nil && postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully logged Custom Food: %s (%s kcal)!\n", name, k)
	} else {
		fmt.Println("Failed to log custom meal.")
	}
	if postResp != nil { postResp.Body.Close() }
}

func logSleep(client *http.Client, bedtime, waketime, quality string) {
	data := url.Values{}
	data.Set("bedtime", bedtime)
	data.Set("waketime", waketime)
	data.Set("quality", quality)
	postResp, err := client.PostForm(dynamicBaseURL+"/api/sleep", data)
	if err == nil && postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully logged sleep from %s to %s!\n", bedtime, waketime)
	} else {
		fmt.Println("Failed to log sleep.")
	}
	if postResp != nil { postResp.Body.Close() }
}

func logCheckin(client *http.Client, dateStr string) {
	req, _ := http.NewRequest("POST", dynamicBaseURL+"/api/checkins?date="+dateStr, nil)
	postResp, err := client.Do(req)
	if err == nil && postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully toggled check-in for %s!\n", dateStr)
	} else {
		fmt.Println("Failed to toggle check-in.")
	}
	if postResp != nil { postResp.Body.Close() }
}

func logWeight(client *http.Client, weight string) {
	data := url.Values{}
	data.Set("weight", weight)
	postResp, err := client.PostForm(dynamicBaseURL+"/api/bodyweight", data)
	if err == nil && postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully logged body weight: %s kg!\n", weight)
	} else {
		fmt.Println("Failed to log body weight.")
	}
	if postResp != nil { postResp.Body.Close() }
}

func saveSession(jar http.CookieJar) {
	u, _ := url.Parse(dynamicBaseURL)
	for _, c := range jar.Cookies(u) {
		if c.Name == "session_id" {
			home, _ := os.UserHomeDir()
			os.WriteFile(filepath.Join(home, cookieFile), []byte(c.Value), 0600)
			return
		}
	}
}

func loadSession(jar http.CookieJar) {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, cookieFile))
	if err == nil {
		u, _ := url.Parse(dynamicBaseURL)
		jar.SetCookies(u, []*http.Cookie{{Name: "session_id", Value: string(data)}})
	}
}

func printUsage() {
	fmt.Println("Gama CLI Companion")
	fmt.Println("Usage:")
	fmt.Println("  gama login <user> <pass>")
	fmt.Println("  gama checkin [YYYY-MM-DD]")
	fmt.Println("  gama weight <kg>")
	fmt.Println("  gama foodslist")
	fmt.Println("  gama food <number_from_catalog>")
	fmt.Println("  gama food \"Name from catalog\"")
	fmt.Println("  gama food \"Custom Meal\" -k 250 -p 30 -c 40 -f 10")
	fmt.Println("  gama sleep [time]")
	fmt.Println("  gama woke [time]")
}
