
-- +migrate Up
CREATE TABLE note_references (
    referent VARCHAR NOT NULL,
    referrer VARCHAR NOT NULL,
    FOREIGN KEY (referent) REFERENCES notes(id),
    FOREIGN KEY (referrer) REFERENCES notes(id)
);
CREATE INDEX idx__note_references__referent ON note_references(referent);
CREATE INDEX idx__note_references__referrer ON note_references(referrer);

-- +migrate Down
DROP INDEX idx__note_references__referent;
DROP INDEX idx__note_references__referrer;
DROP TABLE note_references;
