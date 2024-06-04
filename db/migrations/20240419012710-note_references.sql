
-- +migrate Up
CREATE TABLE note_references (
    id VARCHAR NOT NULL,
    ancestor VARCHAR,
    FOREIGN KEY (ancestor) REFERENCES notes(id),
    FOREIGN KEY (id) REFERENCES notes(id)
);
CREATE INDEX idx__note_references__ancestor ON note_references(ancestor);
CREATE INDEX idx__note_references__id ON note_references(id);

-- +migrate Down
DROP INDEX idx__note_references__ancestor;
DROP INDEX idx__note_references__id;
DROP TABLE note_references;
