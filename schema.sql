CREATE TABLE IF NOT EXISTS organizations (
	id   TEXT PRIMARY KEY,
	name TEXT,
	slug TEXT
);

CREATE TABLE IF NOT EXISTS projects (
	id              TEXT PRIMARY KEY,
	name            TEXT,
	slug            TEXT,
	status          TEXT,
	organization_id TEXT NOT NULL,
	CONSTRAINT fk__projects__organizations
		FOREIGN KEY (organization_id) REFERENCES organizations (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS issues (
	id         TEXT PRIMARY KEY,
	title      TEXT,
	permalink  TEXT,
	has_seen   BOOLEAN,
	first_seen DATETIME,
	last_seen  DATETIME,
	user_count INT,
	level      TEXT,
	status     TEXT,
	type       TEXT,
	project_id TEXT NOT NULL,
	CONSTRAINT fk__issues__projects
		FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE
);

