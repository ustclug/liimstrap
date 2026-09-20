package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func useTestDatabase(t *testing.T) string {
	t.Helper()

	oldWriteDB, oldReadDB := writeDB, readDB
	filename := filepath.Join(t.TempDir(), "state.db")
	var err error
	writeDB, readDB, err = openStateDatabase(filename)
	if !assert.NoError(t, err, "open test database") {
		t.FailNow()
	}
	t.Cleanup(func() {
		assert.NoError(t, readDB.Close(), "close read database")
		assert.NoError(t, writeDB.Close(), "close write database")
		writeDB, readDB = oldWriteDB, oldReadDB
	})
	return filename
}

func TestOpenStateDatabase(t *testing.T) {
	useTestDatabase(t)

	assert.Equal(t, 1, writeDB.Stats().MaxOpenConnections, "write max open connections")
	assert.Equal(t, max(4, runtime.NumCPU()), readDB.Stats().MaxOpenConnections, "read max open connections")

	var strict int
	err := readDB.QueryRow(`SELECT strict FROM pragma_table_list WHERE name = 'clients'`).Scan(&strict)
	assert.NoError(t, err, "query clients table")
	assert.Equal(t, 1, strict, "clients table strict setting")

	// Keep several connections checked out at once so the pool must create each
	// one independently and invoke the driver's connection hook for all of them.
	ctx := context.Background()
	connections := make([]*sql.Conn, 4)
	for i := range connections {
		conn, err := readDB.Conn(ctx)
		if !assert.NoError(t, err, "open read connection %d", i) {
			t.FailNow()
		}
		connections[i] = conn
	}
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()

	for i, conn := range connections {
		assertPragmaText(t, conn, i, "journal_mode", "wal")
		assertPragmaInt(t, conn, i, "busy_timeout", 5000)
		assertPragmaInt(t, conn, i, "synchronous", 1)
		assertPragmaInt(t, conn, i, "foreign_keys", 1)
		assertPragmaInt(t, conn, i, "temp_store", 2)
	}
}

func assertPragmaInt(t *testing.T, conn *sql.Conn, connection int, name string, want int) {
	t.Helper()
	var got int
	err := conn.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&got)
	if assert.NoError(t, err, "connection %d: query PRAGMA %s", connection, name) {
		assert.Equal(t, want, got, "connection %d: PRAGMA %s", connection, name)
	}
}

func assertPragmaText(t *testing.T, conn *sql.Conn, connection int, name, want string) {
	t.Helper()
	var got string
	err := conn.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&got)
	if assert.NoError(t, err, "connection %d: query PRAGMA %s", connection, name) {
		assert.Equal(t, want, got, "connection %d: PRAGMA %s", connection, name)
	}
}

func TestConfigReloadAndHandlersUseDatabaseState(t *testing.T) {
	databaseFile := useTestDatabase(t)
	configFile = filepath.Join(t.TempDir(), "clients.yaml")
	writeConfig := func(contents string) {
		t.Helper()
		if !assert.NoError(t, os.WriteFile(configFile, []byte(contents), 0o600), "write config") {
			t.FailNow()
		}
		if !assert.NoError(t, loadConfig(), "load config") {
			t.FailNow()
		}
	}

	writeConfig(`
timeout: 60
machines:
  - name: First
    mac: aa-bb-cc-dd-ee-ff
  - name: Second
    mac: 11:22:33:44:55:66
`)

	form := url.Values{
		"mac":     {"aa:bb:cc:dd:ee:ff"},
		"version": {"test-version"},
		"uptime":  {"123"},
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.0.2.10:12345"
	recorder := httptest.NewRecorder()
	handleFunc(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code, "POST response body: %s", recorder.Body.String())

	query := httptest.NewRequest(http.MethodGet, "/ip?ip=192.0.2.10", nil)
	recorder = httptest.NewRecorder()
	handleIP(recorder, query)
	assert.Equal(t, http.StatusOK, recorder.Code, "IP lookup response body: %s", recorder.Body.String())
	for _, want := range []string{`"name":"First"`, `"version":"test-version"`, `"uptime":123000000000`} {
		assert.Contains(t, recorder.Body.String(), want, "IP lookup body")
	}

	// Removing a configured machine changes its display classification without
	// discarding the state that was already committed for it.
	writeConfig(`
timeout: 120
machines:
  - name: Second renamed
    mac: 11:22:33:44:55:66
`)
	clients, err := loadClients()
	if !assert.NoError(t, err, "load clients after reload") {
		t.FailNow()
	}
	if !assert.Len(t, clients, 3) {
		return
	}
	assert.Equal(t, "Second renamed", clients[0].Name)
	assert.Empty(t, clients[1].Mac)
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", clients[2].Mac)
	assert.Equal(t, "test-version", clients[2].Version)
	assert.Equal(t, "192.0.2.10", clients[2].IP)
	assert.Equal(t, 123*time.Second, clients[2].Uptime)
	assert.Equal(t, 120*time.Second, time.Duration(aliveTimeout.Load()))
	assert.FileExists(t, databaseFile, "state database")
}

func TestUnknownClientUpdatesFallbackRow(t *testing.T) {
	useTestDatabase(t)
	configFile = filepath.Join(t.TempDir(), "clients.yaml")
	if !assert.NoError(t, os.WriteFile(configFile, []byte("timeout: 60\nmachines: []\n"), 0o600)) {
		t.FailNow()
	}
	if !assert.NoError(t, loadConfig()) {
		t.FailNow()
	}

	form := url.Values{"mac": {"de:ad:be:ef:00:01"}, "version": {"unknown"}, "uptime": {"7"}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "[2001:db8::1]:1234"
	recorder := httptest.NewRecorder()
	handleFunc(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code, "POST response body: %s", recorder.Body.String())

	clients, err := loadClients()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Len(t, clients, 1) {
		return
	}
	assert.Equal(t, "Unknown", clients[0].Name)
	assert.Equal(t, "unknown", clients[0].Version)
	assert.Empty(t, clients[0].Mac)
	assert.Equal(t, "2001:db8::1", clients[0].IP)
	assert.Equal(t, 7*time.Second, clients[0].Uptime)
}
