CREATE TABLE IF NOT EXISTS companies (
    id SERIAL PRIMARY KEY UNIQUE NOT NULL,
    name VARCHAR NOT NULL CHECK (name <> ''),
    key VARCHAR UNIQUE NOT NULL DEFAULT '',
    description TEXT,
    website VARCHAR DEFAULT '',
    logo_url VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL CHECK (email <> ''),
    username VARCHAR UNIQUE NOT NULL CHECK (username <> ''),
    password VARCHAR NOT NULL,
    fullname VARCHAR NOT NULL,
    company_id INTEGER REFERENCES companies(id),
    position VARCHAR DEFAULT '',
    avatar_url VARCHAR DEFAULT ''
);
CREATE TABLE IF NOT EXISTS articles (
    id SERIAL PRIMARY KEY UNIQUE NOT NULL,
    author_id INTEGER NOT NULL REFERENCES users(id),
    cover_url VARCHAR DEFAULT '',
    company_id INTEGER REFERENCES companies(id),
    title VARCHAR NOT NULL CHECK (title <> ''),
    text TEXT NOT NULL CHECK (text <> ''),
    rating INTEGER NOT NULL DEFAULT(0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);