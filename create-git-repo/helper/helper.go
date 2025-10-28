package helper

import (
	"fmt"
	"os"
	s "strings"
)

func WriteToFile(filePath string, content string) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(s.TrimSpace(content)); err != nil {
		panic(err)
	}
	fmt.Printf("Appended content to %s\n", filePath)
}
