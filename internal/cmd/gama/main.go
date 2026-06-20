package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultBaseURL = "http://localhost:8095"
	cookieFile     = ".gama_session"
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

	command := os.Args[1]
	args := os.Args[2:]

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Load saved session
	loadSession(jar)

	switch command {
	case "-login":
		if len(args) < 2 {
			fmt.Println("Usage: gama -login <username> <password>")
			return
		}
		login(client, args[0], args[1])

	case "-foodslist":
		listFoods(client)

	case "-food":
		if len(args) == 0 {
			fmt.Println("Usage: gama -food <item number>")
			return
		}
		idx, _ := strconv.Atoi(args[0])
		logFood(client, idx)

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
		if resp.Header.Get("Location") == "/login.html" || resp.StatusCode == http.StatusFound {
			fmt.Println("Error: Not logged in. Use 'gama -login <user> <pass>'")
		} else {
			fmt.Printf("Error: Status %d\n", resp.StatusCode)
		}
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

func logFood(client *http.Client, pos int) {
	// First fetch catalog to get the name
	resp, err := client.Get(dynamicBaseURL + "/api/foods/catalog")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var items []CatalogFoodItem
	json.NewDecoder(resp.Body).Decode(&items)

	if pos < 1 || pos > len(items) {
		fmt.Println("Invalid food item number. Use 'gama -foodslist' to see options.")
		return
	}
	item := items[pos-1]

	data := url.Values{}
	data.Set("catalog_food", item.Name)
	data.Set("catalog_grams", fmt.Sprintf("%.0f", item.Weight))

	postResp, err := client.PostForm(dynamicBaseURL+"/api/meals", data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer postResp.Body.Close()

	if postResp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully logged %s (%d kcal)!\n", item.Name, item.Calories)
	} else {
		fmt.Printf("Failed to log meal. Status: %d\n", postResp.StatusCode)
	}
}

func saveSession(jar http.CookieJar) {
	u, _ := url.Parse(dynamicBaseURL)
	cookies := jar.Cookies(u)
	for _, c := range cookies {
		if c.Name == "session_id" {
			home, _ := os.UserHomeDir()
			path := filepath.Join(home, cookieFile)
			os.WriteFile(path, []byte(c.Value), 0600)
			return
		}
	}
}

func loadSession(jar http.CookieJar) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, cookieFile)
	data, err := os.ReadFile(path)
	if err == nil {
		u, _ := url.Parse(dynamicBaseURL)
		jar.SetCookies(u, []*http.Cookie{{
			Name:  "session_id",
			Value: string(data),
		}})
	}
}

func printUsage() {
	fmt.Println("Gama CLI Companion")
	fmt.Println("Usage:")
	fmt.Println("  gama -login <user> <pass>   - Log in to sync with web")
	fmt.Println("  gama -foodslist             - List saved foods in catalog")
	fmt.Println("  gama -food <num>            - Log a food from your catalog")
}
