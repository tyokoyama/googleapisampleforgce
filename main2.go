package main

import (
	"fmt"
	"log"
	"net/http"

	// "github.com/garyburd/go-oauth/oauth"
	"github.com/ymotongpoo/go-twitter/twitter"
)

func main() {
	httpClient := &http.Client{}

	client := twitter.NewClient(httpClient)
	client.AddCredential()

	tweet, err := client.HomeTimeline(nil)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(tweet)
}
