CREATE TABLE IF NOT EXISTS team_leaders (
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);

INSERT INTO team_leaders (team_id, user_id) VALUES ('t-eng', 'u-carol');
