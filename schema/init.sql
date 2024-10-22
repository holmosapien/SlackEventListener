CREATE TABLE account (
    id SERIAL PRIMARY KEY,
    email VARCHAR(64) NOT NULL,
    first_name VARCHAR(64) NOT NULL,
    last_name VARCHAR(64) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE client (
    id SERIAL PRIMARY KEY,
    client_id VARCHAR(64) NOT NULL,
    client_secret VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE client_account_membership (
    account_id INTEGER NOT NULL,
    client_id INTEGER NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, client_id)
);

CREATE TABLE oauth_state (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL,
    client_id INTEGER NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    redeemed TIMESTAMP
);

CREATE TABLE team (
    id SERIAL PRIMARY KEY,
    slack_team_id VARCHAR(16) NOT NULL,
    name VARCHAR(64) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (slack_team_id)
);

CREATE TABLE integration (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
    client_id INTEGER NOT NULL,
    slack_user_id VARCHAR(16) NOT NULL,
    access_token VARCHAR(128) NOT NULL,
    app_id VARCHAR(16) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);