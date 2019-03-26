package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/sms", smsEndpoint)
	fmt.Println("listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func smsEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("params: %q\n", r.URL.Query())

	fromCountry := r.URL.Query().Get("FromCountry")
	message := fmt.Sprintf("Greetings, %s", fromCountry)
	w.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>
        %s
    </Message>
</Response>`, message)))
}
