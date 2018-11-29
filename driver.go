package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vmware/dispatch/pkg/events"
	"github.com/vmware/dispatch/pkg/events/driverclient"
)

type validationEvent struct {
	Data struct {
		ValidationCode string `json:"validationCode"`
		ValidationURL  string `json:"validationUrl"`
	} `json:"data"`
	EventType string `json:"eventType"`
	Topic     string `json:"topic"`
}

type validationResponse struct {
	ValidationResponse string `json:"validationResponse"`
}

// debug
var dryRun = flag.Bool("dry-run", false, "Debug, pull messages and do not send Dispatch events")
var org = flag.String("org", "default", "organization of this event driver")
var dispatchEndpoint = flag.String("dispatch-api-endpoint", "localhost:8080", "dispatch server host")
var port = flag.Int("port", 80, "Port to listen on")
var sharedSecret = flag.String("shared-secret", "", "A token or shared secret that the client should pass")

func getDriverClient() driverclient.Client {
	if *dryRun {
		return nil
	}
	token := os.Getenv(driverclient.AuthToken)
	client, err := driverclient.NewHTTPClient(driverclient.WithGateway(*dispatchEndpoint), driverclient.WithToken(token))
	if err != nil {
		log.Fatalf("Error when creating the events client: %s", err.Error())
	}
	log.Println("Event driver initialized.")
	return client
}

func main() {

	flag.Parse()

	client := getDriverClient()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		contentType, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		eType := r.Header.Get("aeg-event-type")
		if contentType == "application/json" && eType == "SubscriptionValidation" {
			// validation handshake

			var e []validationEvent
			bytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.Unmarshal(bytes, &e)
			if len(e) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			resp := json.NewEncoder(w)
			err = resp.Encode(validationResponse{ValidationResponse: e[0].Data.ValidationCode})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if contentType == "application/cloudevents+json" {
			// an actual cloud event
			bytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			evt := events.CloudEvent{}
			err = json.Unmarshal(bytes, &evt)
			if err != nil {
				log.Printf("Error unmarshalling event grid event: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			err = client.SendOne(&evt)
			if err != nil {
				log.Printf("Error sending event grid event: %s", err)
			}
			log.Printf("Sent event successfully %v", evt)
		}
	})

	// Create chan signal
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
	}()

	<-done
}
