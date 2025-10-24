package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Book struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Price    string `json:"price"`
	Imageurl string `json:"image_url"`
}

const PORT string = ":8080"

type Message struct {
	Msg string
}

func jsonMessageByte(msg string) []byte {
	errrMessage := Message{msg}
	byteContent, _ := json.Marshal(errrMessage)
	return byteContent
}

func checkError(err error) {
	if err != nil {
		log.Printf("Error - %v", err)
	}

}

func main() {

	// http://localhost:8080
	http.HandleFunc("/", handleGetBooks)

	// http://localhost:8080/book?id=1
	http.HandleFunc("/book", handleGetBookById)

	// http://localhost:8080/add
	http.HandleFunc("/add", handleAddBook)

	fmt.Printf("App is listening on %v\n", PORT)

	err := http.ListenAndServe(PORT, nil)
	// stop the app is any error to start the server
	if err != nil {
		log.Fatal(err)
	}
}

func handleGetBooks(w http.ResponseWriter, r *http.Request) {
	books, err := getBooks()

	// send server error as response
	if err != nil {
		log.Printf("Server Error %v\n", err)
		w.WriteHeader(500)
		w.Write(jsonMessageByte("Internal server error"))
	} else {
		booksByte, _ := json.Marshal(books)
		w.Write(booksByte)
	}

}

func handleGetBookById(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	// get book id from URL
	bookId := query.Get("id")
	book, err := getBookById(bookId)
	// send server error as response
	if err != nil {
		log.Printf("Server Error %v\n", err)
		w.WriteHeader(500)
		w.Write(jsonMessageByte("Internal server error"))
	} else {
		// check requested book exists or not
		if (Book{}) == book {
			w.Write(jsonMessageByte("Book Not found"))
		} else {
			bookByte, _ := json.Marshal(book)
			w.Write(bookByte)
		}
	}
}

func handleAddBook(w http.ResponseWriter, r *http.Request) {
	// check for post method
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(jsonMessageByte(r.Method + " - Method not allowed"))
	} else {
		// read the body
		newBookByte, err := io.ReadAll(r.Body)
		// check for valid data from client
		if err != nil {
			log.Printf("Client Error %v\n", err)
			w.WriteHeader(400)
			w.Write(jsonMessageByte("Bad Request"))
		} else {
			books, _ := getBooks() // get all books
			var newBooks []Book    // to add new book

			json.Unmarshal(newBookByte, &newBooks)  // new book added
			books = AppendNewBooks(books, newBooks) // Append new books if they are not already available
			// Write all the books in books.json file
			err = saveBooks(books)
			// send server error as response
			if err != nil {
				log.Printf("Server Error %v\n", err)
				w.WriteHeader(500)
				w.Write(jsonMessageByte("Internal server error"))
			} else {
				w.Write(jsonMessageByte("New book added successfully"))
			}

		}
	}
}
