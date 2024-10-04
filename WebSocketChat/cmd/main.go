package main

import (
	"log"
	"net/http"

	"lab1new/chat"
)

func main() {
	log.SetFlags(log.Lshortfile)

	server := chat.NewServer("/entry")
	go server.Listen()

	http.Handle("/", http.FileServer(http.Dir("./web")))

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
