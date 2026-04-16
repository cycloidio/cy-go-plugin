package sentry

import "net/http"

const iframeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<style>
  body { font-family: sans-serif; font-size: 13px; margin: 16px; color: #222; }
  h2 { margin: 0 0 12px; font-size: 15px; }
  h3 { margin: 20px 0 6px; font-size: 13px; color: #555; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 8px; }
  th { background: #f4f4f4; text-align: left; padding: 6px 8px; border-bottom: 2px solid #ddd; }
  td { padding: 5px 8px; border-bottom: 1px solid #eee; }
  .error { color: #c0392b; }
  .warning { color: #e67e22; }
</style>
</head>
<body>
<h2>Sentry Issues (fixture data)</h2>

<h3>YouDeploy API</h3>
<table>
  <tr><th>Title</th><th>Level</th><th>Status</th><th>Has Seen</th><th>Users</th></tr>
  <tr><td>Fixture: NullPointerException in PaymentService</td><td class="error">error</td><td>unresolved</td><td>false</td><td>42</td></tr>
  <tr><td>Fixture: Timeout connecting to database</td><td class="warning">warning</td><td>unresolved</td><td>true</td><td>7</td></tr>
  <tr><td>Fixture: HTTP 502 Bad Gateway from upstream</td><td class="error">error</td><td>unresolved</td><td>false</td><td>13</td></tr>
</table>

<h3>YouDeploy Frontend</h3>
<table>
  <tr><th>Title</th><th>Level</th><th>Status</th><th>Has Seen</th><th>Users</th></tr>
  <tr><td>Fixture: Unhandled promise rejection in auth flow</td><td class="error">error</td><td>unresolved</td><td>false</td><td>28</td></tr>
  <tr><td>Fixture: React render error in Dashboard component</td><td class="error">error</td><td>unresolved</td><td>true</td><td>5</td></tr>
</table>

<h3>YouDeploy Worker</h3>
<table>
  <tr><th>Title</th><th>Level</th><th>Status</th><th>Has Seen</th><th>Users</th></tr>
  <tr><td>Fixture: Memory usage exceeded threshold</td><td class="warning">warning</td><td>unresolved</td><td>true</td><td>3</td></tr>
  <tr><td>Fixture: Job queue backed up: retries exhausted</td><td class="error">error</td><td>unresolved</td><td>false</td><td>19</td></tr>
  <tr><td>Fixture: Deadlock detected in task scheduler</td><td class="error">error</td><td>unresolved</td><td>false</td><td>8</td></tr>
  <tr><td>Fixture: Worker failed to connect to Redis</td><td class="warning">warning</td><td>unresolved</td><td>true</td><td>11</td></tr>
</table>
</body>
</html>`

func IframeHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(iframeHTML))
}
