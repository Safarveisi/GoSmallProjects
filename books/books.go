package main

import (
	"encoding/json"
	"os"
)

func getBooks() ([]Book, error) {
	books := []Book{}
	booksByte, err := os.ReadFile("./books.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(booksByte, &books)
	if err != nil {
		return nil, err
	}
	return books, nil
}

func getBookById(id string) (Book, error) {
	books, err := getBooks()
	var requestedBook Book

	if err != nil {
		return Book{}, err
	}

	for _, book := range books {
		if book.Id == id {
			requestedBook = book
		}
	}

	return requestedBook, nil
}

// save books to books.json file
func saveBooks(books []Book) error {

	// converting into bytes for writing into a file
	booksBytes, err := json.Marshal(books)

	checkError(err)

	err = os.WriteFile("./books.json", booksBytes, 0644)

	return err

}

func AppendNewBooks(books, newBooks []Book) []Book {
	// Build a set of IDs that already exist in `books`.
	existing := make(map[string]struct{}, len(books))
	for _, b := range books {
		existing[b.Id] = struct{}{}
	}

	// Walk through newBooks and append only the “new” ones.
	for _, nb := range newBooks {
		if _, ok := existing[nb.Id]; !ok {
			books = append(books, nb)
			existing[nb.Id] = struct{}{}
		}
	}
	// If nothing was added, `books` is exactly the original slice.
	return books
}
