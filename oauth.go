package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type SlackTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType string `json:"token_type"`
	Scope string `json:"scope"`
	BotUserId string `json:"bot_user_id"`
	AppId string `json:"app_id"`
	Team struct {
		Name string `json:"name"`
		Id string `json:"id"`
	} `json:"team"`
	Enterprise struct {
		Name string `json:"name"`
		Id string `json:"id"`
	} `json:"enterprise"`
	AuthedUser struct {
		Id string `json:"id"`
		Scope string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType string `json:"token_type"`
	} `json:"authed_user"`
}

func GetSlackLink(context *EventListenerContext, accountId int, clientId int) (string, error) {
	var stateId int
	var clientInfo *SlackClientInfo
	var err error

	stateId, err = SaveOAuthState(context, accountId, clientId)

	if err != nil {
		errorMessage := fmt.Sprintf("Could not save the OAuth state for account_id=%d, client_id=%d: %s", accountId, clientId, err.Error())

		return "", errors.New(errorMessage)
	}

	stateContents := map[string]int{
		"state_id": stateId,
		"account_id": accountId,
		"client_id": clientId,
	}

	stateJson, err := json.Marshal(stateContents)

	if err != nil {
		errorMessage := "Could not marshal the state: " + err.Error()

		return "", errors.New(errorMessage)
	}

	state := base64.StdEncoding.EncodeToString(stateJson)

	userScopes := []string{
		"channels:history",
		"channels:read",
		"groups:history",
		"groups:read",
		"im:history",
		"im:read",
		"mpim:history",
		"mpim:read",
		"team:read",
		"users:read",
	}

	userScope := ""

	for i, scope := range userScopes {
		if i > 0 {
			userScope += ","
		}

		userScope += scope
	}

	clientInfo, err = GetSlackClientInfo(context, accountId, clientId)

	if err != nil {
		errorMessage := fmt.Sprintf("Could not retrieve client info for client_id=%d", clientId)

		return "", errors.New(errorMessage)
	}

	params := url.Values{}

	params.Add("client_id", clientInfo.SlackClientId)
	params.Add("scope", "")
	params.Add("user_scope", userScope)
	params.Add("redirect_uri", "https://slack.holmosapien.com/authorization")
	params.Add("state", state)

	url := "https://slack.com/oauth/v2/authorize?" + params.Encode()

	return url, nil
}

func ExchangeCodeForToken(context *EventListenerContext, code string, state string) (*SlackTokenResponse, error) {
	var oauthState *OAuthState
	var token *SlackTokenResponse

	// Decode the state to find out which account and client we're dealing with.
	//
	// The state is base64 encoded JSON that looks like this:
	//
	// {
	//     "state_id": 1,
	//     "account_id": 1,
	//     "client_id": 1
	// }

	stateJson, err := base64.StdEncoding.DecodeString(state)

	if err != nil {
		errorMessage := "Could not decode the state: " + err.Error()

		return nil, errors.New(errorMessage)
	}

	var stateContents map[string]int

	err = json.Unmarshal(stateJson, &stateContents)

	if err != nil {
		errorMessage := "Could not unmarshal the state: " + err.Error()

		return nil, errors.New(errorMessage)
	}

	stateId := stateContents["state_id"]
	accountId := stateContents["account_id"]
	clientId := stateContents["client_id"]

	// Now we can get the Slack client ID that was used to generate the state.

	oauthState, err = GetOAuthState(context, stateId, accountId, clientId)

	if err != nil {
		errorMessage := fmt.Sprintf("Could not retrieve client info for account_id=%d, client_id=%d", accountId, clientId)

		return nil, errors.New(errorMessage)
	}

	if oauthState == nil {
		errorMessage := fmt.Sprintf("Could not find the OAuth state for account_id=%d, client_id=%d", accountId, clientId)

		return nil, errors.New(errorMessage)
	}

	token, err = GetSlackToken(context, oauthState, code)

	if err != nil {
		errorMessage := "Could not exchange the token: " + err.Error()

		return nil, errors.New(errorMessage)
	}

	// Now that we have the token, redeem the state so it can't be used again.

	err = RedeemOAuthState(context, oauthState)

	if err != nil {
		log.Println("Error redeeming state: " + err.Error())
	}

	return token, nil
}

func GetSlackToken(context *EventListenerContext, oauthState *OAuthState, code string) (*SlackTokenResponse, error) {
	request, err := http.NewRequest("GET", "https://slack.com/api/oauth.v2.access", nil)

    if err != nil {
        return nil, err
    }

    q := request.URL.Query()

    q.Add("client_id", oauthState.SlackClientId)
	q.Add("client_secret", oauthState.ClientSecret)
	q.Add("code", code)

    request.URL.RawQuery = q.Encode()

	client := &http.Client{}

	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	tokenResponse, err := HandleTokenResponse(context, oauthState, body)

	if err != nil {
		return nil, err
	}

	return tokenResponse, nil
}

func HandleTokenResponse(context *EventListenerContext, oauthState *OAuthState, body []byte) (*SlackTokenResponse, error) {
	var tokenResponse SlackTokenResponse
	var teamId int

	var err error

	err = json.Unmarshal(body, &tokenResponse)

	if err != nil {
		return nil, err
	}

	slackTeamId := tokenResponse.Team.Id
	teamName := tokenResponse.Team.Name

	teamId, err = InsertTeam(context, slackTeamId, teamName)

	if err != nil {
		return nil, err
	}

	accountId := oauthState.AccountId
	clientId := oauthState.ClientId

	_, err = InsertIntegration(context, accountId, clientId, teamId, &tokenResponse)

	if err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}