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
	defaultBaseURL    = "http://localhost:8095"
	cookieFile = ".gama_session"
)

var dynamicBaseURL string

func init() {
	dynamicBaseURL = os.Getenv("GAMA_URL")
	if dynamicBaseURL == "" {
		dynamicBaseURL = defaultBaseURL
	}
}

type FocusTask struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
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
	case "login":
		if len(args) < 2 {
			fmt.Println("Usage: gama login <username> <password>")
			return
		}
		login(client, args[0], args[1])

	case "addtask":
		if len(args) == 0 {
			fmt.Println("Usage: gama addtask <task name>")
			return
		}
		addTask(client, strings.Join(args, " "))

	case "list":
		listTasks(client)

	case "done":
		if len(args) == 0 {
			fmt.Println("Usage: gama done <task number>")
			return
		}
		idx, _ := strconv.Atoi(args[0])
		actionTask(client, idx, "done")

	case "clear":
		if len(args) == 0 {
			fmt.Println("Usage: gama clear <task number>")
			return
		}
		idx, _ := strconv.Atoi(args[0])
		actionTask(client, idx, "clear")

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

func addTask(client *http.Client, title string) {
	task := FocusTask{Title: title}
	body, _ := json.Marshal(task)
	resp, err := client.Post(dynamicBaseURL+"/api/focus", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("Task added: %s\n", title)
	} else if resp.StatusCode == http.StatusFound || resp.Header.Get("Location") == "/login.html" {
		fmt.Println("Error: Not logged in. Use 'gama login <user> <pass>'")
	} else {
		fmt.Printf("Error: Status %d\n", resp.StatusCode)
	}
}

func listTasks(client *http.Client) {
	resp, err := client.Get(dynamicBaseURL + "/api/focus")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.Header.Get("Location") == "/login.html" || resp.StatusCode == http.StatusFound {
			fmt.Println("Error: Not logged in. Use 'gama login <user> <pass>'")
		} else {
			fmt.Printf("Error: Status %d\n", resp.StatusCode)
		}
		return
	}

	var tasks []FocusTask
	json.NewDecoder(resp.Body).Decode(&tasks)

	fmt.Println("Focus List:")
	for i, t := range tasks {
		status := "[ ]"
		if t.Completed {
			status = "[x]"
		}
		fmt.Printf("%d. %s %s\n", i+1, status, t.Title)
	}
}

func actionTask(client *http.Client, pos int, action string) {
	// Need to get ID first
	resp, _ := client.Get(dynamicBaseURL + "/api/focus")
	var tasks []FocusTask
	json.NewDecoder(resp.Body).Decode(&tasks)
	resp.Body.Close()

	if pos < 1 || pos > len(tasks) {
		fmt.Println("Invalid task number")
		return
	}
	task := tasks[pos-1]

	var req *http.Request
	if action == "done" {
		task.Completed = true
		body, _ := json.Marshal(task)
		req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/focus/%d", dynamicBaseURL, task.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/focus/%d", dynamicBaseURL, task.ID), nil)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if action == "done" {
			fmt.Println("Task marked as done!")
		} else {
			fmt.Println("Task cleared!")
		}
	} else {
		fmt.Printf("Error: Status %d\n", resp.StatusCode)
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
	fmt.Println("Gama CLI Focus List (HTTP Mode)")
	fmt.Println("Usage:")
	fmt.Println("  gama login <user> <pass>   - Log in to sync with web")
	fmt.Println("  gama addtask <name>        - Add task to Focus List")
	fmt.Println("  gama list                  - Show Focus List")
	fmt.Println("  gama done <num>            - Mark task as done")
	fmt.Println("  gama clear <num>           - Remove task")
}
