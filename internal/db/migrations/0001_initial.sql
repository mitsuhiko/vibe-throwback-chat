-- Initial database schema for ThrowBackChat

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nickname TEXT NOT NULL,
    is_serv BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    topic TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS ops (
    user_id INTEGER NOT NULL,
    channel_id INTEGER NOT NULL,
    granted_by_user_id INTEGER NOT NULL,
    granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, channel_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id),
    FOREIGN KEY (granted_by_user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER,
    user_id INTEGER NOT NULL,
    sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    message TEXT NOT NULL,
    is_passive BOOLEAN DEFAULT FALSE,
    event TEXT NOT NULL, -- 'joined', 'left', 'announcement', 'nick_change', 'message'
    nickname TEXT NOT NULL,
    FOREIGN KEY (channel_id) REFERENCES channels(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL UNIQUE,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create the default ChanServ user
INSERT OR IGNORE INTO users (id, nickname, is_serv) VALUES (1, 'ChanServ', TRUE);