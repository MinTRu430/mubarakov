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



CREATE TABLE IF NOT EXISTS elections (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,  -- 'created', 'open', 'closed', 'counted'
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    rsa_m TEXT NOT NULL,  
    rsa_e INTEGER NOT NULL,
    rsa_d TEXT NOT NULL   
);

CREATE TABLE IF NOT EXISTS election_votes (
    id SERIAL PRIMARY KEY,
    election_id INTEGER NOT NULL REFERENCES elections(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ciphertext TEXT NOT NULL, -- f_i = t_i^e mod m
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    UNIQUE (election_id, user_id)
);

CREATE TABLE IF NOT EXISTS election_results (
    election_id INTEGER PRIMARY KEY REFERENCES elections(id) ON DELETE CASCADE,
    yes_votes INTEGER NOT NULL,
    no_votes INTEGER NOT NULL,
    abstain_votes INTEGER NOT NULL,
    R TEXT NOT NULL,            -- произведение q_i (без 2 и 3)
    total_voters INTEGER NOT NULL,
    F TEXT NOT NULL,            -- произведение всех f_i (ciphertext)
    Q TEXT NOT NULL,            -- дешифрованное произведение t_i (до разложения)
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
