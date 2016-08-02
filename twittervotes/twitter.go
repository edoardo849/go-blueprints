package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/go-oauth/oauth"
	"github.com/joeshaw/envdecode"
)

var conn net.Conn

func dial(netw, addr string) (net.Conn, error) {

	// checks if the connection already exist
	if conn != nil {
		// close the connection if it exist
		conn.Close()
		conn = nil
	}

	netc, err := net.DialTimeout(netw, addr, 5*time.Second)

	if err != nil {
		return nil, err
	}
	conn = netc

	return netc, err
}

var reader io.ReadCloser

// close the connection to Twitter at any time
func closeConn() {
	log.Println("Closing the connection to Twitter")
	if conn != nil {
		conn.Close()
		log.Println("The connection is closed")
	}

	if reader != nil {
		reader.Close()
	}
}

var (
	authClient *oauth.Client
	creds      *oauth.Credentials
)

// this func will only be called once as sets the vars
func setupTwitterAuth() {
	var ts struct {
		ConsumerKey    string `env:"SP_TWITTER_KEY,required"`
		ConsumerSecret string `env:"SP_TWITTER_SECRET,required"`
		AccessToken    string `env:"SP_TWITTER_ACCESSTOKEN,required"`
		AccessSecret   string `env:"SP_TWITTER_ACCESSSECRET,required"`
	}

	if err := envdecode.Decode(&ts); err != nil {
		log.Fatalln(err)
	}

	creds = &oauth.Credentials{
		Token:  ts.AccessToken,
		Secret: ts.AccessSecret,
	}

	authClient = &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  ts.ConsumerKey,
			Secret: ts.ConsumerSecret,
		},
	}
}

var (
	authSetupOnce sync.Once
	httpClient    *http.Client
)

func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {

	// Ensure that the init code only gets run once despite the number of times
	// we call `makeRequest`
	authSetupOnce.Do(func() {
		setupTwitterAuth()
		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: dial,
			},
		}
	})

	formEnc := params.Encode()

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization", authClient.AuthorizationHeader(creds, "POST", req.URL, params))

	return httpClient.Do(req)
}

type tweet struct {
	Text string
}

// readfromTwitter receives a receive-only channel.
func readfromTwitter(votes chan<- string) {

	options, err := loadOptions()
	if err != nil {
		log.Println("failed to load options:", err)
		return
	}

	u, err := url.Parse("https://stream.twitter.com/1.1/statuses/filter.json")
	if err != nil {
		log.Println("creating filter request failed:", err)
		return
	}

	query := make(url.Values)
	query.Set("track", strings.Join(options, ","))

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(query.Encode()))

	if err != nil {
		log.Println("creating filter request failed: ", err)
		return
	}

	resp, err := makeRequest(req, query)

	if err != nil {
		log.Println("making request failed: ", err)
		return
	}

	reader := resp.Body
	decoder := json.NewDecoder(reader)

	// infinte loop because the connection, while being open, will return
	// stream of data that we'll need to process
	for {
		var tw tweet

		if err := decoder.Decode(&tw); err != nil {
			break
		}

		// code here for Graph data

		// loop for every option
		for _, option := range options {
			if strings.Contains(
				strings.ToLower(tw.Text),
				strings.ToLower(option),
			) {
				log.Println(tw.Text)
				log.Println("vote: ", option)

				votes <- option
			}
		}

	}

}

func startTwitterStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	stoppedchan := make(chan struct{}, 1)
	go func() {

		defer func() {
			stoppedchan <- struct{}{}
		}()

		// unlless signled to stopchan, we are reconneting to Twitter
		// after 10 seconds that the connection has dropped.
		for {
			select {
			case <-stopchan:
				log.Println("stopping Twitter...")
				return
			default:
				log.Println("querying Twitter...")
				readfromTwitter(votes)
				log.Println(" (waiting)")
				time.Sleep(10 * time.Second) // wait 10 seconds before reconnecting
			}
		}
	}()

	return stoppedchan
}
