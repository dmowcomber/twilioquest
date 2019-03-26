package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var todoList = map[int]string{}

func main() {
	http.HandleFunc("/sms", smsEndpoint)
	fmt.Println("listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func smsEndpoint(w http.ResponseWriter, r *http.Request) {
	textMessage := r.URL.Query().Get("Body")
	textMessageLower := strings.ToLower(textMessage)
	fmt.Printf("request body: %q\n", textMessage)

	todoCount := len(todoList)
	if strings.HasPrefix(textMessageLower, "add ") {
		addThing := textMessage[4:]
		todoList[todoCount] = addThing
		w.Write([]byte(lazyTwimlMessage(fmt.Sprintf("added %s", addThing))))
		return
	}
	if strings.HasPrefix(textMessageLower, "remove ") {
		remove := strings.TrimPrefix(textMessageLower, "remove ")
		removeIndex, err := strconv.Atoi(remove)
		if err != nil {
			w.Write([]byte(lazyTwimlMessage(fmt.Sprintf("unable to remove %q", remove))))
			return
		}
		delete(todoList, removeIndex)
		w.Write([]byte(lazyTwimlMessage(fmt.Sprintf("removed %d", removeIndex))))
		return
	}
	if textMessageLower == "list" {
		var respBody string
		var newline string
		for k, v := range todoList {
			respBody = respBody + newline + strconv.Itoa(k) + ". " + v
			newline = "\n"
		}
		w.Write([]byte(lazyTwimlMessage(respBody)))
		return
	}
}

func lazyTwimlMessage(message string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>
        %s
    </Message>
</Response>`, message)
}
