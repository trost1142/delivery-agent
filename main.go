package main

import ( 
	"context"
	"fmt"
	kafka "github.com/segmentio/kafka-go"
	"encoding/json"
	"strings"
	"regexp"
	"log"
	"os"
	"time"
	"net/http"
	"io/ioutil"
)

const (
	topic		= "postback.delivery"
	broker1Address 	= "localhost:9092"
)

type Message struct {
	Method string `json:"endpoint_method"`
	Url string `json:"endpoint_url"`
	StartTime string `json:"start_time"`
	Data map[string]interface{}
}


func main() {
	// Create context for Kafka Consumer
	ctx := context.Background()
	
	// Start consuming data from Kafka Consumer
	consume(ctx)
    
}

func consume(ctx context.Context) {
	// Create log file for delivery times and responses
	file, file_err := os.OpenFile("/var/log/delivery-agent.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	log.SetOutput(file)
	
	if file_err != nil {
		log.Fatalf("error opening file: %v", file_err)
	}

	// Create Kafka Consumer
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker1Address},
		Topic: topic,
	})

	var message Message
	for {
		
		// Read in messages through Kafka Consumer
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			panic("could not read message " + err.Error())
		}

		// Decode Kafka message into Message structure
		json.Unmarshal(msg.Value, &message)

		// Add data values from Kafka message into request URL
		filled_url := process_url(message.Data, message.Url)
		
		// Get delivery time in UTC
		loc, _ := time.LoadLocation("UTC")	
		currentTime := time.Now().In(loc)
		
		// Convert and format start time (from PHP service) to UTC datetime
		const layout = "2006-01-02 15:04:05.999999"
		start_time, _ := time.Parse(layout, message.StartTime)
		
		// Get delivery time and start time in milliseconds
		milliseconds_start := start_time.UnixNano() / int64(time.Millisecond)
		milliseconds_delivery := currentTime.UnixNano() / int64(time.Millisecond)

		// Calculate delivery flight time
		diff := milliseconds_delivery - milliseconds_start
		
		// Send request to processed url
		response, error := make_request(message.Method, filled_url, 3)

		// Catch internal network errors
		if error != nil {
			log.Println("Failed to get response from url: ", filled_url, " using method: ", message.Method)
		} else {
			// Get response datetime in UTC
			response_time := time.Now().In(loc)
			
			// Calculate total flight time in milliseconds
			milliseconds_response := response_time.UnixNano() / int64(time.Millisecond)
			diff_response := milliseconds_response - milliseconds_start
			
			// Decode request response body and status code
			body, body_err := ioutil.ReadAll(response.Body)
			if body_err != nil {
				log.Fatalln(body_err)
			}

			body_string := string(body)
			statusCode := response.StatusCode

			// Log all information to configured log file
			log.Println(currentTime, " ", diff, " ", response_time, " ", diff_response, " ", statusCode, " ", body_string)
		}
	}

}

func make_request(method string, url string, attempts int) (*http.Response, error) {
	currentAttempts := 0
	
	// Attempt to retry url if error occurrs or non-OK status code returned from original request until allowed attempts runs out
	for {
		currentAttempts++
		// Send GET request
		resp, err := http.Get(url)

		// If no errors and OK HTTP response code returned, return full responsee
		// Else if network error or response code not OK and there are still attempts left, pause for 5 seconds and retry
		// Else return error response
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		} else if (err != nil || resp.StatusCode != http.StatusOK) && attempts != currentAttempts {
			//resp.Body.Close()
			fmt.Println("Retrying request: ", attempts - currentAttempts, " attemps remaining") 
			time.Sleep(5 * time.Second)
		} else {
			return resp, err
		}

	}

}


func process_url(data map[string]interface{}, originalUrl string) string {
	url := originalUrl
	//fmt.Println("Data", data)
	//fmt.Println("URL", originalUrl)

	for k, v := range data {
		//fmt.Println("key", k)
		//fmt.Println("value", v)
		replace := "{" + k + "}"
		url = strings.Replace(url, replace, v.(string), -1)
		//fmt.Println("New URL", url)
	}

	final_url := trim_missing_vars(url)
	//fmt.Println(url_temp)

	return final_url
}


func trim_missing_vars(originalURL string) string {
	missing_regex := regexp.MustCompile(`{.*}`)
	url_trimmed :=  missing_regex.ReplaceAllString(originalURL, `null`)

	return url_trimmed
}
