package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"path"
	"time"
)

type JsonPlaceHolder struct {
	Id    int16  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func worker(id int, jobs <-chan Url, results chan<- Url) {
	for url := range jobs {
		fmt.Println("worker", id, "started working on", url)
		post, err := url.fetch()
		fmt.Println("worker", id, "finished working on", url)
		if err != nil {
			results <- url
		} else {
			url.success = true
			url.post = post
			results <- url
		}
	}
}

type Url struct {
	url     string
	success bool
	post    *JsonPlaceHolder
}

func (url *Url) fetch() (*JsonPlaceHolder, error) {
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

	var p JsonPlaceHolder
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &p, nil
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
			fmt.Printf("🔹 %d – id=%d title=%q\n", i+1, p.post.Id, p.post.Title)
		} else {
			fmt.Printf("❌ error getting post: %s", p.url)
		}
	}

}
