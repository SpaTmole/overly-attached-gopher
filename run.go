package main

import (
	"encoding/json"
	"fmt"
	"github.com/subosito/gotenv"
	"github.com/yanatan16/golang-instagram/instagram"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type AccessTokenResponse struct {
	AccessToken string `json:access_token`
}

var authCodeAccessTokenPool map[string]string

// var pool map[string]chan (int)
var CLIENT_ID string
var CLIENT_SECRET string
var REDIRECT_URI string

func init() {
	gotenv.Load()
	authCodeAccessTokenPool = make(map[string]string)
	CLIENT_ID = os.Getenv("INSTAGRAM_CLIENT_ID")
	CLIENT_SECRET = os.Getenv("INSTAGRAM_CLIENT_SECRET")
	REDIRECT_URI = os.Getenv("INSTAGRAM_REDIRECT")
}

func startWatchingForLikes(accessToken string) {
	fmt.Println("Access Token: ", accessToken)
	api := instagram.New(CLIENT_ID, CLIENT_SECRET, accessToken, true)
	if _, err := api.VerifyCredentials(); err != nil {
		fmt.Println("Invalid Instagram credentials.")
		fmt.Println(err)
		fmt.Println(accessToken)
		return
	}
	// TODO:
	// REMEMBER RECENT MEDIA
	// REMEBER LIKES OF EACH POST
	// ADD TO REFRESH NEW MEDIA
	// ADD REFRESH NEW LIKES
	// GO TO NEW LIKE-POSTED-USERS AND LIKE THEIR LAST n POSTS
	params := url.Values{}
	params.Set("count", "10")
	if resp, err := api.GetUserRecentMedia("self", params); err != nil {
		fmt.Println(err)
		return
	} else {
		if len(resp.Medias) == 0 { // [sic]
			fmt.Println("I should have some sort of media posted on instagram!")
			return
		}
		for _, media := range resp.Medias {
			fmt.Printf(
				"Post #(%s): '%s' \nWith %d comments and %d likes. (url: %s)\n",
				media.Id,
				media.Caption,
				media.Comments.Count,
				media.Likes.Count,
				media.Link,
			)
			fmt.Println("Liked by:")
			for _, user := range media.Likes.Data {
				fmt.Println(user.Username)
			}
		}
	}
}

func viewHandler(writer http.ResponseWriter, request *http.Request) {
	urlRedirect := fmt.Sprintf(
		"https://api.instagram.com/oauth/authorize/?client_id=%s&response_type=code&scope=likes+public_content&redirect_uri=%s",
		CLIENT_ID,
		REDIRECT_URI,
	)
	http.Redirect(writer, request, urlRedirect, http.StatusFound)
}

func authHandler(writer http.ResponseWriter, request *http.Request) {
	errorReason := request.FormValue("error_reason")
	if errorReason != "" {
		errorDescription := request.FormValue("error_description")
		http.Error(writer, errorDescription, http.StatusBadRequest)
		return
	}
	authCode := request.FormValue("code")
	if authCode != "" {
		resp, err := http.PostForm(
			"https://api.instagram.com/oauth/access_token/",
			url.Values{
				"client_id":     {CLIENT_ID},
				"client_secret": {CLIENT_SECRET},
				"code":          {authCode},
				"redirect_uri":  {REDIRECT_URI},
				"grant_type":    {"authorization_code"},
			})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		jsonResp := make(map[string]interface{})
		err = json.Unmarshal(body, &jsonResp)
		if err != nil {
			panic(err)
		}
		fmt.Println(jsonResp)
		accessToken := jsonResp["access_token"]
		if accessToken != nil {
			go startWatchingForLikes(jsonResp["access_token"].(string))
		} else {
			http.Error(writer, fmt.Sprintf("%s", body), http.StatusBadRequest)
			return
		}
		fmt.Fprintf(
			writer,
			"<html><body> We've starting looking after your likes. :) </body></html>",
		)
		return
	}

	http.Error(writer, "Access Token shouldn't be empty.", http.StatusBadRequest)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		fn(writer, request)
	}
}

func main() {
	if os.Getenv("INSTAGRAM_CLIENT_ID") == "" || os.Getenv("INSTAGRAM_CLIENT_SECRET") == "" {
		fmt.Println("Emtpy Instagram credentials. Check the .env file.")
		os.Exit(1)
	}
	http.HandleFunc("/", makeHandler(viewHandler))
	http.HandleFunc("/auth/confirm", makeHandler(authHandler))
	fmt.Println("Listening connections on http://127.0.0.1:8080 ...")
	http.ListenAndServe(":8080", nil)
	fmt.Println("Success!")
}
