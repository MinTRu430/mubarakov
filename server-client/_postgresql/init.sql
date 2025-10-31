CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY, 
    login VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    rsa_signature TEXT NOT NULL,
    code_word VARCHAR(100),
    code_word_live_time TIMESTAMP,
    tg_login VARCHAR(255) NOT NULL UNIQUE,
    tg_code VARCHAR(100),
    tg_chat_id BIGINT
);

CREATE TABLE IF NOT EXISTS dh_temp (
    login VARCHAR(255) PRIMARY KEY,
    a_value TEXT,
    a_private TEXT,
    p_value TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
