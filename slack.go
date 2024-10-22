package main

import (
    "encoding/json"
    "errors"
    "fmt"
)

type SlackEvent struct {
    Type string `json:"type"`
    Token string `json:"token"`
    Challenge string `json:"challenge"` // Only present in URL verification requests.
    TeamId string `json:"team_id"`
    ApiAppId string `json:"api_app_id"`
    EventContext string `json:"event_context"`
    Event struct {
        Type string `json:"type"`
    }
}

type URLVerificationResponse struct {
    Challenge string `json:"challenge"`
}

type SlackEventResponse struct {
    Message string `json:"message"`
}

func GetEventHandler(eventType string) func(*EventListenerContext, *SlackEvent) (interface{}, error) {
    switch eventType {
    case "url_verification":
        return HandleUrlVerification
    case "event_callback":
        return HandleEventCallbackEvent
    default:
        return HandleUnknownEvent
    }
}

func ProcessSlackEvent(context *EventListenerContext, body []byte) (interface{}, error) {
    var response interface{}
    var err error

    // Before we go parsing the event, let's just dump it straight into the database.

    err = InsertRawSlackEvent(context, body)

    // Now we can parse the event and see if we need to do something special with it.

    var event SlackEvent

    err = json.Unmarshal(body, &event)

    if err != nil {
        errorMessage := "Could not parse the request body: " + err.Error()

        return nil, errors.New(errorMessage)
    }

    eventType := event.Type

    handler := GetEventHandler(eventType)
    response, err = handler(context, &event)

    if err != nil {
        errorMessage := "An error occurred while processing the event: " + err.Error()

        return nil, errors.New(errorMessage)
    }

    return response, nil
}

func HandleUrlVerification(context *EventListenerContext, event *SlackEvent) (interface{}, error) {
    fmt.Println("Handling URL verification request: " + event.Challenge)

    return &URLVerificationResponse{
        Challenge: event.Challenge,
    }, nil
}

func HandleEventCallbackEvent(context *EventListenerContext, event *SlackEvent) (interface{}, error) {
    fmt.Println("Handling event_callback event: " + event.Event.Type)

    return &SlackEventResponse{
        Message: "Message received.",
    }, nil
}

func HandleUnknownEvent(context *EventListenerContext, event *SlackEvent) (interface{}, error) {
    fmt.Println("Handling unknown event type: " + event.Type)

    return &SlackEventResponse{
        Message: "Unknown event type.",
    }, nil
}