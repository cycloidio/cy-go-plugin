package sentry

import (
	"database/sql"
	"fmt"
)

func Seed(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM organizations`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if _, err := db.Exec(`INSERT INTO organizations (id, name, slug) VALUES ('1', 'Youdeploy Org', 'youdeploy-org')`); err != nil {
		return err
	}

	projects := []struct{ id, name, slug string }{
		{"101", "YouDeploy API", "youdeploy-api"},
		{"102", "YouDeploy Frontend", "youdeploy-frontend"},
		{"103", "YouDeploy Worker", "youdeploy-worker"},
	}
	for _, p := range projects {
		if _, err := db.Exec(
			`INSERT INTO projects (id, name, slug, status, organization_id) VALUES (?, ?, ?, 'active', '1')`,
			p.id, p.name, p.slug,
		); err != nil {
			return err
		}
	}

	issues := []struct {
		id, title, level, projectID string
		hasSeen                     bool
		userCount                   int
	}{
		{"101-001", "Fixture: NullPointerException in PaymentService", "error", "101", false, 42},
		{"101-002", "Fixture: Timeout connecting to database", "warning", "101", true, 7},
		{"101-003", "Fixture: HTTP 502 Bad Gateway from upstream", "error", "101", false, 13},
		{"102-001", "Fixture: Unhandled promise rejection in auth flow", "error", "102", false, 28},
		{"102-002", "Fixture: React render error in Dashboard component", "error", "102", true, 5},
		{"103-001", "Fixture: Memory usage exceeded threshold", "warning", "103", true, 3},
		{"103-002", "Fixture: Job queue backed up: retries exhausted", "error", "103", false, 19},
		{"103-003", "Fixture: Deadlock detected in task scheduler", "error", "103", false, 8},
		{"103-004", "Fixture: Worker failed to connect to Redis", "warning", "103", true, 11},
	}
	for _, i := range issues {
		firstSeen := `datetime('now', '-7 days')`
		if i.hasSeen {
			firstSeen = `datetime('now', '-1 day')`
		}
		if _, err := db.Exec(fmt.Sprintf(`
			INSERT INTO issues (id, title, permalink, has_seen, first_seen, last_seen, user_count, level, status, type, project_id)
			VALUES (?, ?, ?, ?, %s, datetime('now'), ?, ?, 'unresolved', 'error', ?)`,
			firstSeen,
		), i.id, i.title,
			"https://sentry.io/organizations/youdeploy-org/issues/"+i.id+"/",
			i.hasSeen, i.userCount, i.level, i.projectID,
		); err != nil {
			return err
		}
	}
	return nil
}

func Clear(db *sql.DB) error {
	for _, table := range []string{"issues", "projects", "organizations"} {
		if _, err := db.Exec(`DELETE FROM ` + table); err != nil {
			return err
		}
	}
	return nil
}
