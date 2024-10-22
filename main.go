package main

import (
	"github.com/gin-gonic/gin"
	"os"
	"strconv"
)

func main() {
	// Initialize the context.

	context := InitializeContext()

	// Set up the router.

    router := gin.Default()

	router.GET("/redirect-link", context.RedirectToSlack)
    router.GET("/authorization", context.ReceiveAuthorizationCode)
	router.POST("/event", context.ReceiveSlackEvent)

	// Start the server.
	// If a certificate and key are not provided, the server will start in HTTP mode.

	certificatePath := os.Getenv("EVENT_LISTENER_CERTIFICATE_PATH")
	keyPath := os.Getenv("EVENT_LISTENER_KEY_PATH")

	if certificatePath != "" && keyPath != "" {
    	router.RunTLS(":18075", certificatePath, keyPath)
	} else {
		router.Run(":18075")
	}
}

func (context *EventListenerContext) RedirectToSlack(c *gin.Context) {
	accountIdParam := c.Query("account_id")
	clientIdParam := c.Query("client_id")

	var accountId int
	var clientId int

	var err error

	if accountIdParam == "" {
		c.JSON(400, gin.H{
			"error": "The account_id query parameter is required.",
		})

		return
	}

	accountId, err = strconv.Atoi(accountIdParam)

	if err != nil {
		c.JSON(400, gin.H{
			"error": "The account_id query parameter must be an integer.",
		})

		return
	}

	if clientIdParam == "" {
		c.JSON(400, gin.H{
			"error": "The client_id query parameter is required.",
		})

		return
	}

	clientId, err = strconv.Atoi(clientIdParam)

	if err != nil {
		c.JSON(400, gin.H{
			"error": "The client_id query parameter must be an integer.",
		})

		return
	}

	slackLink, err := GetSlackLink(context, accountId, clientId)

	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
	}

	c.Redirect(302, slackLink)
}

func (context *EventListenerContext) ReceiveAuthorizationCode(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	_, err := ExchangeCodeForToken(context, code, state)

	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(204, gin.H{})
}

func (context *EventListenerContext) ReceiveSlackEvent(c *gin.Context) {
	var body []byte
	var response interface{}
	var err error

	body, err = c.GetRawData()

	if err != nil {
		c.JSON(500, gin.H{
			"error": "Could not read the request body.",
		})
	}

	response, err = ProcessSlackEvent(context, body)

	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(200, response)
}