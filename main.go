package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"path"
	s "strings"
	"time"
)

type UserPost struct {
	PostId   int16  `json:"id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Comments *[]PostComments
}

type PostComments struct {
	PostId int16  `json:"postId"`
	Id     int16  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Body   string `json:"body"`
}

func worker(id int, jobs <-chan Url, results chan<- Url) {
	for url := range jobs {
		fmt.Println("worker", id, "started working on", url)
		post, err1 := url.fetchPost()
		comments, err2 := url.fetchComments()
		fmt.Println("worker", id, "finished working on", url)
		if err1 == nil {
			url.success = true
			url.post = post
			if err2 == nil {
				url.post.Comments = comments
			}
		}

		results <- url
	}
}

type Url struct {
	url     string
	success bool
	post    *UserPost
}

func (url *Url) fetchPost() (*UserPost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// New request that inherits the caller's context (so timeout/cancel works)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	// Use the default client – it has a built‑in transport and connection pool.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var p UserPost
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &p, nil
}

func (url *Url) fetchComments() (*[]PostComments, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// New request that inherits the caller's context (so timeout/cancel works)
	urlComments := s.Join([]string{url.url, "comments"}, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlComments, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	// Use the default client – it has a built‑in transport and connection pool.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var c []PostComments
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &c, nil
}

func main() {

	// Absolute path to the file containing urls to be queried
	filename := flag.String("filename",
		"/home/user/file.txt", "The absolute path to a file of urls")
	countWorkers := flag.Int("nw", 2, "Number of parallel jobs")

	flag.Parse()

	if !path.IsAbs(*filename) {
		panic("'filename' should be an absolute path to a file")
	}

	if *countWorkers < 0 {
		panic("Numnber of workers cannot be nagative")
	}

	urls, err := readLines(filename)
	if err != nil {
		panic(err)
	}

	numJobs := len(urls)

	jobs := make(chan Url, numJobs)
	resps := make(chan Url, numJobs)

	for w := 1; w <= *countWorkers; w++ {
		go worker(w, jobs, resps)
	}

	for _, url := range urls {
		jobs <- url
	}

	results := make([]Url, 0, numJobs)

	for a := 1; a <= numJobs; a++ {
		results = append(results, <-resps)
	}

	for i, p := range results {
		if p.success {
			fmt.Printf("🔹 %d - id=%d title=%q\n", i+1, p.post.PostId, p.post.Title)
		} else {
			fmt.Printf("❌ error getting post: %s", p.url)
		}
	}

	fmt.Println()

	for _, p := range results {
		if p.success && p.post.Comments != nil {
			fmt.Printf("Following people commented on Post id %d:\n", p.post.PostId)
			for i, c := range *p.post.Comments {
				fmt.Printf("\t(%d) 🔹Name: %s\n", i, c.Name)
			}
		} else {
			fmt.Printf("❌ error getting comments: %s\n", p.url)
		}
	}

}
