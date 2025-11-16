package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Andrea-Cavallo/espresso/pkg/espresso"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func main() {
	// Quick Start - Just 3 lines of code!

	// 1. Create a client
	client := espresso.NewDefaultClient[User]()

	// 2. Make a request
	response, err := client.Request("https://jsonplaceholder.typicode.com/users/1").Get(context.Background())

	// 3. Handle the response
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: %s (%s)\n", response.Data.Name, response.Data.Email)
}
