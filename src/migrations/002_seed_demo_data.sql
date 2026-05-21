-- Demo users
INSERT INTO users (id, username, display_name, email) VALUES
    ('u-alice', 'alice', 'Alice Johnson', 'alice@example.com'),
    ('u-bob',   'bob',   'Bob Smith',     'bob@example.com'),
    ('u-carol', 'carol', 'Carol Davis',   'carol@example.com'),
    ('u-admin', 'admin', 'Administrator', 'admin@example.com')
ON CONFLICT (id) DO NOTHING;

-- Platform usernames
INSERT INTO user_platform_usernames (user_id, platform, username) VALUES
    ('u-alice', 'GITHUB', 'alice-gh'),
    ('u-bob',   'GITHUB', 'bob-gh'),
    ('u-carol', 'GITHUB', 'carol-gh')
ON CONFLICT (user_id, platform) DO NOTHING;

-- Accounts (password: "secret" hashed with bcrypt cost 10)
-- $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
INSERT INTO user_accounts (id, username, password_hash, roles, user_id) VALUES
    ('a-alice', 'alice', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '{TEAM_MEMBER}', 'u-alice'),
    ('a-bob',   'bob',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '{TEAM_MEMBER}', 'u-bob'),
    ('a-carol', 'carol', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '{TEAM_MEMBER,TEAM_LEADER}', 'u-carol'),
    ('a-admin', 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '{ADMIN}', 'u-admin')
ON CONFLICT (id) DO NOTHING;

-- Team
INSERT INTO teams (id, name) VALUES
    ('t-eng', 'Engineering')
ON CONFLICT (id) DO NOTHING;

-- Team members
INSERT INTO team_members (team_id, user_id) VALUES
    ('t-eng', 'u-alice'),
    ('t-eng', 'u-bob'),
    ('t-eng', 'u-carol')
ON CONFLICT (team_id, user_id) DO NOTHING;

-- Sample repositories
INSERT INTO repositories (id, name, full_name, url, platform) VALUES
    ('r-ct',  'contribution-tracker', 'myorg/contribution-tracker', 'https://github.com/myorg/contribution-tracker', 'GITHUB'),
    ('r-api', 'api-service',          'myorg/api-service',          'https://github.com/myorg/api-service',          'GITHUB')
ON CONFLICT (id) DO NOTHING;

-- Assign repos to team
INSERT INTO team_repositories (team_id, repo_id) VALUES
    ('t-eng', 'r-ct'),
    ('t-eng', 'r-api')
ON CONFLICT (team_id, repo_id) DO NOTHING;
