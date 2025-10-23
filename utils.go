package main

import (
	"bufio"
	"fmt"
	"os"
)

func readLines(filename *string) ([]Url, error) {
	f, err := os.Open(*filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var lines []Url
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// ignore blank lines (useful if the file has a trailing newline)
		if line != "" {
			lines = append(lines, Url{url: line})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	return lines, nil
}
