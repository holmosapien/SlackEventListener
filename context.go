package main

import (
    "github.com/jackc/pgx/v5/pgxpool"
    "os"
)


type EventListenerContext struct {
    DatabaseHostname  string
    DatabasePort      string
    DatabaseUsername  string
    DatabasePassword  string
    DatabaseName      string
    DatabasePool      *pgxpool.Pool
}

func InitializeContext() *EventListenerContext {
    hostname := os.Getenv("EVENT_LISTENER_PG_HOSTNAME")
    port := os.Getenv("EVENT_LISTENER_PG_PORT")
    username := os.Getenv("EVENT_LISTENER_PG_USERNAME")
    password := os.Getenv("EVENT_LISTENER_PG_PASSWORD")
    database := os.Getenv("EVENT_LISTENER_PG_DATABASE")

    context := EventListenerContext{
        DatabaseHostname:  hostname,
        DatabasePort:      port,
        DatabaseUsername:  username,
        DatabasePassword:  password,
        DatabaseName:      database,
    }

    pool, err := CreateDatabasePool(&context)

    if err != nil {
        panic(err)
    }

    context.DatabasePool = pool

    return &context
}