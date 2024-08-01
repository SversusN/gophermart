BEGIN TRANSACTION;


CREATE TABLE IF NOT EXISTS users
(
    id       SERIAL PRIMARY KEY,
    login    TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    "current" FLOAT NOT NULL DEFAULT 0,
    withdrawal FLOAT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS accruals
(
    order_num   BIGINT PRIMARY KEY,
    user_id     INT  NOT NULL,
    status      TEXT NOT NULL            DEFAULT 'NEW',
    amount      REAL                     DEFAULT 0,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS withdrawals
(
    order_num    BIGINT PRIMARY KEY,
    user_id      INT NOT NULL,
    amount       REAL                     DEFAULT 0,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users (id)
);

COMMIT TRANSACTION;