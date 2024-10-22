package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "time"
)

type OAuthState struct {
    Id int
    AccountId int
    ClientId int
    SlackClientId string
    ClientSecret string
}

type SlackClientInfo struct {
    Id int
    SlackClientId string
    ClientSecret string
    Name string
}

func CreatePoolConfig(c *EventListenerContext) (*pgxpool.Config) {
    const defaultMaxConns = int32(4)
    const defaultMinConns = int32(0)
    const defaultMaxConnLifetime = time.Hour
    const defaultMaxConnIdleTime = time.Minute * 30
    const defaultHealthCheckPeriod = time.Minute
    const defaultConnectTimeout = time.Second * 5

    databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
        c.DatabaseUsername,
        c.DatabasePassword,
        c.DatabaseHostname,
        c.DatabasePort,
        c.DatabaseName)

    dbConfig, err := pgxpool.ParseConfig(databaseUrl)

    if err != nil {
        log.Fatal("Failed to create PGX config: ", err)
    }

    dbConfig.MaxConns = defaultMaxConns
    dbConfig.MinConns = defaultMinConns
    dbConfig.MaxConnLifetime = defaultMaxConnLifetime
    dbConfig.MaxConnIdleTime = defaultMaxConnIdleTime
    dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
    dbConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout

    dbConfig.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
        return true
    }

    dbConfig.AfterRelease = func(c *pgx.Conn) bool {
        return true
    }

    dbConfig.BeforeClose = func(c *pgx.Conn) {
        return
    }

    return dbConfig
}

func CreateDatabasePool(c *EventListenerContext) (*pgxpool.Pool, error) {
    pool, err := pgxpool.NewWithConfig(context.Background(), CreatePoolConfig(c))

    if err != nil {
        log.Fatal("Failed to create PGX pool: ", err)

        return nil, err
    }

    return pool, nil
}

func GetSlackClientInfo(c *EventListenerContext, accountId int, clientId int) (*SlackClientInfo, error) {
    var clientInfo SlackClientInfo

    err := c.DatabasePool.QueryRow(context.Background(), `
        SELECT
            id,
            client_id,
            client_secret
        FROM
            client
        WHERE
            id = $1`, clientId).Scan(&clientInfo.Id, &clientInfo.SlackClientId, &clientInfo.ClientSecret)

    if err != nil {
        errorMessage := fmt.Sprintf("Error getting client_id=%d: %s", clientId, err.Error())

        return nil, errors.New(errorMessage)
    }

    return &clientInfo, nil
}

func SaveOAuthState(c *EventListenerContext, accountId int, clientId int) (int, error) {
    var stateId int

    err := c.DatabasePool.QueryRow(context.Background(), `
        INSERT INTO
            oauth_state
        (
            account_id,
            client_id,
            created
        ) VALUES (
            $1,
            $2,
            NOW()
        )
        RETURNING
            id`, accountId, clientId).Scan(&stateId)

    if err != nil {
        fmt.Println("Error saving state: " + err.Error())

        return 0, err
    }

    return stateId, err
}

func GetOAuthState(c *EventListenerContext, stateId int, accountId int, clientId int) (*OAuthState, error) {
    var state OAuthState

    err := c.DatabasePool.QueryRow(context.Background(), `
        SELECT
            os.id,
            os.account_id,
            c.id AS client_id,
            c.client_id AS slack_client_id,
            c.client_secret
        FROM
            oauth_state os
        JOIN
            client c ON os.client_id = c.id
        WHERE
            os.id = $1 AND
            os.account_id = $2 AND
            os.client_id = $3 AND
            os.redeemed IS NULL
        ORDER BY
            os.created DESC
        LIMIT
            1`, stateId, accountId, clientId).Scan(&state.Id, &state.AccountId, &state.ClientId, &state.SlackClientId, &state.ClientSecret)

    if err != nil {
        errorMessage := fmt.Sprintf("Error getting team ID by account_id=%d, client_id=%d", accountId, clientId)

        return nil, errors.New(errorMessage)
    }

    return &state, nil
}

func RedeemOAuthState(c *EventListenerContext, oauthState *OAuthState) error {
    _, err := c.DatabasePool.Exec(context.Background(), `
        UPDATE
            oauth_state
        SET
            redeemed = NOW()
        WHERE
            id = $1`, oauthState.Id)

    if err != nil {
        errorMessage := fmt.Sprintf("Error redeeming state_id=%d", oauthState.Id)

        return errors.New(errorMessage)
    }

    return nil
}

func InsertTeam(c *EventListenerContext, slackTeamId string, teamName string) (int, error) {
    var teamId int

    err := c.DatabasePool.QueryRow(context.Background(), `
        INSERT INTO
            team
        (
            slack_team_id,
            name,
            created
        ) VALUES (
            $1,
            $2,
            NOW()
        )
        ON CONFLICT
            (slack_team_id)
        DO UPDATE
            SET name = $3
        RETURNING
            id`, slackTeamId, teamName, teamName).Scan(&teamId)

    if err != nil {
        errorMessage := fmt.Sprintf("Error inserting slack_team_id=%s, name=%s", slackTeamId, teamName)

        return 0, errors.New(errorMessage)
    }

    return teamId, nil
}

func InsertIntegration(c *EventListenerContext, accountId int, clientId int, teamId int, tokenResponse *SlackTokenResponse) (int, error) {
    slackUserId := tokenResponse.AuthedUser.Id
    userToken := tokenResponse.AuthedUser.AccessToken
    appId := tokenResponse.AppId

    var integrationId int

    err := c.DatabasePool.QueryRow(context.Background(), `
        INSERT INTO
            integration
        (
            account_id,
            client_id,
            team_id,
            slack_user_id,
            access_token,
            app_id,
            created
        ) VALUES (
            $1,
            $2,
            $3,
            $4,
            $5,
            $6,
            NOW()
        )
        RETURNING
            id`, accountId, clientId, teamId, slackUserId, userToken, appId).Scan(&integrationId)

    if err != nil {
        errorMessage := fmt.Sprintf("Error inserting integration for account_id=%d, client_id=%d, team_id=%d", accountId, clientId, teamId)

        return 0, errors.New(errorMessage)
    }

    return integrationId, nil
}

func InsertRawSlackEvent(c *EventListenerContext, body []byte) error {
    _, err := c.DatabasePool.Exec(context.Background(), `
        INSERT INTO
            raw_event
        (
            event,
            created
        ) VALUES (
             $1,
            NOW()
        )`, body)

    if err != nil {
        fmt.Println("Error inserting raw event: " + err.Error())

        return err
    }

    return err
}