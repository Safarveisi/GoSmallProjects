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
	"strings"
	"time"
)

var personalAccessTokenURL = "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a_personal_access_token"

var defaultFiles = []string{
	".gitignore",
	"README.md",
	"LICENSE.md",
	".github/workflows/ci.yaml",
	"tests/",
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
	createRemote := flag.Bool("create-remote", false, "Whether to create remote repository on GitHub")

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
		base, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		fmt.Printf("Creating project skeleton in %s\n", base)
		makeProjectSkeleton(base)

		// Write sample content to some files
		helper.WriteToFile(".github/workflows/ci.yaml", workflowContent)
		helper.WriteToFile("README.md", fmt.Sprintf("# %s\n\nThis is the README for the %s repository.\n", *repoName, *repoName))

		fmt.Printf("Created local repository %s at %s\n", *repoName, time.Now().Format("2006-01-02 15:04:05"))
		// Create a file including the repo creation time
		f, err := os.Create("REPO_CREATED_AT.txt")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		f.WriteString(fmt.Sprintf("Repository %s created at %s\n", *repoName, time.Now().Format("2006-01-02 15:04:05")))

		// Create remote repository on GitHub if requested
		if *createRemote {
			fmt.Println("Creating remote repository on GitHub...")
			token, exists := os.LookupEnv("GITHUB_TOKEN")
			if !exists {
				fmt.Fprintf(os.Stderr,
					"GITHUB_TOKEN environment variable is not set. Please see %s\n", personalAccessTokenURL)
				os.Exit(1)
			}
			err = createRemoteRepo(*repoName, token)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating remote repository: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Skipping remote repository creation on GitHub.")
			return
		}
	}
}

func makeProjectSkeleton(baseDir string) error {
	for _, rel := range defaultFiles {
		// normalise the path (remove a possible leading "./")
		rel = strings.TrimPrefix(rel, "./")

		abs := filepath.Join(baseDir, rel)

		// A directory is indicated by a trailing slash (POSIX) or by the OS‑specific separator.
		// We also treat an entry that already exists and is a directory the same way.
		if strings.HasSuffix(rel, string(os.PathSeparator)) || strings.HasSuffix(rel, "/") {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return fmt.Errorf("creating directory %q: %w", abs, err)
			}
			continue
		}

		// Ensure the parent directory exists before we try to create the file.
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensuring parent directory %q: %w", dir, err)
		}

		// "Touch" the file - create it if it does not exist, keep it unchanged otherwise.
		f, err := os.OpenFile(abs, os.O_RDONLY|os.O_CREATE, 0o644)
		if err != nil {
			return fmt.Errorf("creating file %q: %w", abs, err)
		}
		f.Close()
	}

	return nil
}

func createRemoteRepo(repoName, token string) error {
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
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Remote repository created successfully on GitHub.")
	} else {
		return fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}
	return nil
}
