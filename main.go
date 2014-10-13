package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/mrjones/oauth"
)

const (
	consumerKey    = "uTcNQaAkSd2bgAjAyrSId5lES"
	consumerSecret = "wklLdTsTxlpcxATLMYLBo82tBdRXFtiplfzx3PjnST5ageUC2m"
)

func main() {
	c := oauth.NewConsumer(
		consumerKey,
		consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		},
	)

	requestToken, url, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("(1) Go to: " + url)
	fmt.Println("(2) Grant access, you should get back a verification code.")
	fmt.Println("(3) Enter that verification code here: ")

	verificationCode := ""
	fmt.Scanln(&verificationCode)

	accessToken, err := c.AuthorizeToken(requestToken, verificationCode)
	if err != nil {
		log.Fatal(err)
	}

	// response, err := c.Get(
	// 	"https://api.twitter.com/1.1/statuses/home_timeline.json",
	// 	map[string]string{"count": "1"},
	// 	accessToken)
	response, err := c.Get(
		"https://api.twitter.com/1.1/statuses/home_timeline.json",
		nil,
		accessToken)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	fmt.Println("The newest item in your home timeline is: " + base64.StdEncoding.EncodeToString(bits))

}
