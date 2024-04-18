-- +migrate Up
PRAGMA foreign_keys = on;
CREATE TABLE users (
    id varchar PRIMARY KEY,
    alias_id varchar(15) NOT NULL UNIQUE,
    password varchar NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX uniq_idx__users__alias_id ON users(alias_id);

CREATE TABLE notes (
    id varchar PRIMARY KEY,
    user_id varchar NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx__notes__user_id ON notes(user_id);

CREATE TABLE note_revisions (
    id varchar PRIMARY KEY,
    note_id varchar NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    FOREIGN KEY (note_id) REFERENCES notes(id)
);

CREATE INDEX idx__note_revisions__note_id ON note_revisions(note_id);

-- +migrate Down
DROP INDEX idx__note_revisions__note_id;
DROP TABLE note_revisions;
DROP INDEX idx__notes__user_id;
DROP TABLE notes;
DROP INDEX uniq_idx__users__alias_id;
DROP TABLE users;
