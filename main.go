package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "test")
	})

	fmt.Println("Serveur démarré sur le port 3030...")
	fmt.Println("http://localhost:3030/")
	http.ListenAndServe(":3030", nil)
}
