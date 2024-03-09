package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

type Tweet struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	config := oauth1.NewConfig(os.Getenv("CONSUMER_KEY"), os.Getenv("CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	twitterID := os.Getenv("TWITTER_ID")

	for {
		twitterLikesURL := fmt.Sprintf("https://api.twitter.com/2/users/%s/liked_tweets", twitterID)

		// Fetch likes
		resp, err := httpClient.Get(twitterLikesURL)
		if err != nil {
			fmt.Printf("Error fetching likes: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			return
		}

		var likedTweets struct {
			Data []Tweet `json:"data"`
		}
		err = json.Unmarshal(body, &likedTweets)
		if err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			return
		}

		if len(likedTweets.Data) == 0 {
			fmt.Println("No more likes to delete.")
			break
		}

		// Delete likes
		for _, tweet := range likedTweets.Data {
			deleteURL := fmt.Sprintf("https://api.twitter.com/2/users/%s/likes/%s", twitterID, tweet.ID)
			req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
			if err != nil {
				fmt.Printf("Error creating request for tweet %s: %v\n", tweet.ID, err)
				continue
			}
			deleteResp, err := httpClient.Do(req)
			if err != nil {
				fmt.Printf("Error deleting like for tweet %s: %v\n", tweet.ID, err)
				continue
			}
			fmt.Printf("Deleted like for tweet: \"%s\", status code: %d\n", tweet.Text, deleteResp.StatusCode)

			if deleteResp.StatusCode == http.StatusTooManyRequests {
				fmt.Println("Rate limit exceeded. Waiting for reset...")
				resetTime := getRateLimitResetTime(deleteResp)
				timeToWait := time.Until(resetTime)
				fmt.Printf("Waiting for %v\n", timeToWait)
				time.Sleep(timeToWait)
			}

			deleteResp.Body.Close()
		}
	}
}

func getRateLimitResetTime(resp *http.Response) time.Time {
	resetTimestamp, err := strconv.ParseInt(resp.Header.Get("x-rate-limit-reset"), 10, 64)
	if err != nil {
		fmt.Printf("Error parsing rate limit reset time: %v\n", err)
		return time.Now().Add(time.Minute) // Default to waiting 1 minute if parsing fails
	}
	return time.Unix(resetTimestamp, 0)
}
