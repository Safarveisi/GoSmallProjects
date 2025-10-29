package main

import (
	"bytes"
	"create-git-repo/helper"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var defaultFiles = []string{
	".gitignore",
	"README.md",
	"LICENSE.md",
	".github/workflows/ci.yaml",
}

var workflowContent = `
# This is a sample GitHub Actions workflow
name: CI

on:
  push:
    branches: [ master ]
  pull_request:
    branches: [ master ]

jobs:
  build:
    runs-on: ubuntu-latest
`

func main() {

	repoName := flag.String("repo-name", "dummy", "Name of the repository")
	repoUser := flag.String("repo-user", "autocommitbot", "User name for git commits")
	repoEmail := flag.String("repo-email", "autocommitbot@example.com", "Email for git commits")

	flag.Parse()

	// Create directory for the new repo
	os.Mkdir(*repoName, 0755)
	// Change working directory to the new repo
	os.Chdir(*repoName)

	if _, err := os.Stat(".git"); err == nil {
		fmt.Println("A git repository already exists. Skipping git init.")
	} else {
		fmt.Println("> git init")
		err := exec.Command("git", "init").Run()
		if err != nil {
			panic(err)
		}
		fmt.Println("Initialized empty Git repository.")

		fmt.Println("> git branch -M master")
		err = exec.Command("git", "branch", "-M", "master").Run()
		if err != nil {
			panic(err)
		}
		fmt.Println("Renamed default branch to 'master'.")

		fmt.Println("Setting user name and email for git commits.")
		// Set user name and email for git commits
		fmt.Printf("> git config user.name %s\n", *repoUser)
		err = exec.Command("git", "config", "user.name", *repoUser).Run()
		if err != nil {
			panic(err)
		}
		fmt.Printf("> git config user.email %s\n", *repoEmail)
		err = exec.Command("git", "config", "user.email", *repoEmail).Run()
		if err != nil {
			panic(err)
		}
		fmt.Println("Configured username and email.")

		fmt.Printf("> git remote add origin git@github.com:%s/%s.git\n", *repoUser, *repoName)
		err = exec.Command(
			"git",
			"remote",
			"add",
			"origin",
			fmt.Sprintf("git@github.com:%s/%s.git", *repoUser, *repoName),
		).Run()
		if err != nil {
			panic(err)
		}
		fmt.Println("Added remote origin.")

		// Create default files
		CreateDefaultFiles()
		helper.WriteToFile(".github/workflows/ci.yaml", workflowContent)
		helper.WriteToFile("README.md", fmt.Sprintf("# %s\n\nThis is the README for the %s repository.\n", *repoName, *repoName))

		fmt.Printf("Created repository %s at %s\n", *repoName, time.Now().Format("2006-01-02 15:04:05"))
		// Create a file including the repo creation time
		f, err := os.Create("REPO_CREATED_AT.txt")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		f.WriteString(fmt.Sprintf("Repository %s created at %s\n", *repoName, time.Now().Format("2006-01-02 15:04:05")))

		token, exists := os.LookupEnv("GITHUB_TOKEN")
		if !exists {
			fmt.Fprintln(os.Stderr, "GITHUB_TOKEN environment variable is not set")
			os.Exit(1)
		}

		CreateRemoteRepo(*repoName, token)
	}
}

func CreateDefaultFiles() {
	for _, file := range defaultFiles {
		// Creates any necessary parent directories
		dir := filepath.Dir(file)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				panic(err)
			}
		}
		// Create the file
		f, err := os.Create(file)
		if err != nil {
			panic(err)
		}
		f.Close()
		fmt.Printf("Created file: %s\n", file)
	}
	fmt.Println("Created default files: README.md, .gitignore, ...")
}

func CreateRemoteRepo(repoName, token string) {
	payload := map[string]interface{}{
		"name":        repoName,
		"description": "This is a newly created repository",
		"homepage":    "https://github.com",
		"private":     false,
		"is_template": false,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost,
		"https://api.github.com/user/repos",
		bytes.NewReader(body))

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Remote repository created successfully on GitHub.")
	} else {
		fmt.Fprintf(os.Stderr, "GitHub returned status %d\n", resp.StatusCode)
	}
}
