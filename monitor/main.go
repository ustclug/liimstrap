package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

const (
	sqliteDriverName  = "sqlite3_liims_monitor"
	connectionPragmas = `
PRAGMA temp_store = memory;
`
	stateSchema = `
CREATE TABLE IF NOT EXISTS clients (
    mac         TEXT PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL,
    version     TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    last_seen   INTEGER NOT NULL DEFAULT 0,
    uptime      INTEGER NOT NULL DEFAULT 0,
    configured  INTEGER NOT NULL DEFAULT 0 CHECK (configured IN (0, 1)),
    sort_order  INTEGER NOT NULL DEFAULT 0
) STRICT;
CREATE INDEX IF NOT EXISTS clients_display_order
    ON clients (configured DESC, sort_order);
CREATE INDEX IF NOT EXISTS clients_ip ON clients (ip);
`
)

func init() {
	sql.Register(sqliteDriverName, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			_, err := conn.Exec(connectionPragmas, nil)
			return err
		},
	})
}

type Config struct {
	Timeout  int64 `json:"timeout"`
	Machines []struct {
		Name string `json:"name"`
		Mac  string `json:"mac"`
	} `json:"machines"`
}

type ClientInfo struct {
	Name    string        `json:"name"`
	Version string        `json:"version"`
	Mac     string        `json:"mac"`
	IP      string        `json:"ip"`
	Time    time.Time     `json:"time"`
	Uptime  time.Duration `json:"uptime"`
}

func (ci *ClientInfo) Status() string {
	if time.Since(ci.Time) > time.Duration(aliveTimeout.Load()) {
		if ci.Mac == "" {
			return "unknown"
		}
		return "down"
	}
	return "ok"
}

func (ci *ClientInfo) TimeStr() string {
	if ci.Time.IsZero() {
		return "Never"
	}
	return ci.Time.Format(time.DateTime)
}

// Modified from https://gist.github.com/harshavardhana/327e0577c4fed9211f65
func (ci *ClientInfo) UptimeStr() string {
	d := ci.Uptime
	if d == 0 {
		return ""
	}
	days := int64(d.Hours() / 24)
	hours := int64(math.Mod(d.Hours(), 24))
	minutes := int64(math.Mod(d.Minutes(), 60))
	seconds := int64(math.Mod(d.Seconds(), 60))
	if days < 1 {
		return fmt.Sprintf("%d:%02d:%02d",
			hours, minutes, seconds)
	}
	daysPlural := "s"
	if days == 1 {
		daysPlural = ""
	}
	return fmt.Sprintf("%d day%s, %d:%02d:%02d",
		days, daysPlural, hours, minutes, seconds)
}

var NonMacChars = regexp.MustCompile("[^0-9a-f]")

func NormalizeMac(mac string) string {
	mac = NonMacChars.ReplaceAllString(strings.ToLower(mac), "")
	if len(mac) != 12 {
		return mac
	}
	s := mac[0:2]
	for i := 2; i < 12; i += 2 {
		s += ":" + mac[i:i+2]
	}
	return s
}

var (
	configFile      string
	listenHost      string
	listenPort      int
	useIPFromHeader bool
	dumpTemplate    bool
	stateFile       string

	aliveTimeout atomic.Int64
	writeDB      *sql.DB
	readDB       *sql.DB

	//go:embed index.html
	indexTemplateStr string
	indexTemplate    template.Template = *template.Must(template.New("index").Parse(indexTemplateStr))
)

func openStateDatabase(filename string) (*sql.DB, *sql.DB, error) {
	separator := "?"
	if strings.Contains(filename, "?") {
		separator = "&"
	}
	dsn := filename + separator + strings.Join([]string{
		"_txlock=immediate",
		"_journal_mode=WAL",
		"_busy_timeout=5000",
		"_synchronous=NORMAL",
		"_cache_size=1000000000",
		"_foreign_keys=true",
	}, "&")

	writeDB, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open write database: %w", err)
	}
	writeDB.SetMaxOpenConns(1)
	if err := writeDB.Ping(); err != nil {
		writeDB.Close()
		return nil, nil, fmt.Errorf("open write database: %w", err)
	}
	if _, err := writeDB.Exec(stateSchema); err != nil {
		writeDB.Close()
		return nil, nil, fmt.Errorf("initialize database: %w", err)
	}

	readDB, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		writeDB.Close()
		return nil, nil, fmt.Errorf("open read database: %w", err)
	}
	readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))
	if err := readDB.Ping(); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, nil, fmt.Errorf("open read database: %w", err)
	}
	return writeDB, readDB, nil
}

func loadConfig() error {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	var config Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return err
	}
	tx, err := writeDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE clients SET configured = 0`); err != nil {
		return err
	}
	const upsertClient = `
INSERT INTO clients (mac, name, configured, sort_order) VALUES (?, ?, 1, ?)
ON CONFLICT (mac) DO UPDATE SET
    name = excluded.name,
    configured = 1,
    sort_order = excluded.sort_order
`
	hasUnknown := false
	for i, m := range config.Machines {
		m.Mac = NormalizeMac(m.Mac)
		if m.Mac == "" {
			hasUnknown = true
		}
		if _, err := tx.Exec(upsertClient, m.Mac, m.Name, i); err != nil {
			return err
		}
	}
	configuredClients := len(config.Machines)
	if !hasUnknown {
		if _, err := tx.Exec(upsertClient, "", "Unknown", configuredClients); err != nil {
			return err
		}
		configuredClients++
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	aliveTimeout.Store(int64(time.Duration(config.Timeout) * time.Second))
	log.Printf("Loaded configuration, total %d configured clients", configuredClients)
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanClient(row rowScanner) (ClientInfo, error) {
	var client ClientInfo
	var lastSeen int64
	var uptime int64
	err := row.Scan(
		&client.Name,
		&client.Version,
		&client.Mac,
		&client.IP,
		&lastSeen,
		&uptime,
	)
	if err != nil {
		return ClientInfo{}, err
	}
	if lastSeen != 0 {
		client.Time = time.Unix(0, lastSeen)
	}
	client.Uptime = time.Duration(uptime)
	return client, nil
}

func loadClients() ([]ClientInfo, error) {
	rows, err := readDB.Query(`
SELECT name, version, mac, ip, last_seen, uptime
FROM clients
ORDER BY configured DESC, sort_order, rowid
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []ClientInfo
	for rows.Next() {
		client, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func handleSignal(chSig <-chan os.Signal) {
	for sig := range chSig {
		switch sig {
		case syscall.SIGHUP:
			log.Printf("Received SIGHUP\n")
			if err := loadConfig(); err != nil {
				log.Printf("Cannot reload config: %v", err)
			}
		case syscall.SIGQUIT:
			log.Printf("Received SIGQUIT\n")
			os.Exit(0)
		}
	}
}

func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(ip, "[]")
}

func getClientIP(r *http.Request) string {
	if useIPFromHeader {
		if ip := normalizeIP(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
	}
	return normalizeIP(r.RemoteAddr)
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		clients, err := loadClients()
		if err != nil {
			log.Printf("Cannot load clients: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// Render HTML list
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)

		err = indexTemplate.Execute(w, clients)
		if err != nil {
			log.Printf("Error rendering index template: %v", err)
		}
	} else if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "text/plain")
		r.ParseForm()
		mac := NormalizeMac(r.PostFormValue("mac"))
		version := r.PostFormValue("version")
		uptimeStr := r.PostFormValue("uptime")
		if mac == "" || version == "" || uptimeStr == "" {
			http.Error(w, "OK", http.StatusBadRequest)
			return
		}
		uptime, err := strconv.Atoi(uptimeStr)
		if err != nil {
			log.Printf("Invalid uptime %#v: %v", uptimeStr, err)
			http.Error(w, "OK", http.StatusBadRequest)
			return
		}

		_, err = writeDB.ExecContext(r.Context(), `
UPDATE clients
SET ip = ?, last_seen = ?, version = ?, uptime = ?
WHERE mac = COALESCE((SELECT mac FROM clients WHERE mac = ?), '')
`, getClientIP(r), time.Now().UnixNano(), version, int64(time.Duration(uptime)*time.Second), mac)
		if err != nil {
			log.Printf("Cannot update client %s: %v", mac, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		http.Error(w, "OK", http.StatusOK)
	} else {
		http.Error(w, "OK", http.StatusMethodNotAllowed)
	}
}

// query location from IP
func handleIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query().Get("ip")
	if q == "" {
		http.Error(w, "Where's your IP?", http.StatusBadRequest)
		return
	}

	d, err := scanClient(readDB.QueryRowContext(r.Context(), `
SELECT name, version, mac, ip, last_seen, uptime
FROM clients
WHERE ip = ?
ORDER BY configured DESC, sort_order, rowid
LIMIT 1
`, q))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, fmt.Sprintf("IP %s not found", q), http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Cannot query IP %s: %v", q, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(d); err != nil {
		log.Printf("Cannot encode client for IP %s: %v", q, err)
	}
}

func main() {
	flag.StringVar(&configFile, "c", "clients.yaml", "YAML config of clients")
	flag.StringVar(&listenHost, "h", "127.0.0.1", "host to listen on")
	flag.IntVar(&listenPort, "p", 2999, "port to listen on")
	flag.BoolVar(&useIPFromHeader, "x", false, "Use X-Real-IP header as client IP")
	flag.StringVar(&stateFile, "s", "/var/lib/liims-monitor/state.db", "state database file")
	flag.BoolVar(&dumpTemplate, "t", false, "dump template and exit")
	flag.Parse()
	if dumpTemplate {
		os.Stdout.Write([]byte(indexTemplateStr))
		return
	}

	// $JOURNAL_STREAM is set by systemd v231+
	if _, ok := os.LookupEnv("JOURNAL_STREAM"); ok {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	var err error
	writeDB, readDB, err = openStateDatabase(stateFile)
	if err != nil {
		log.Fatalf("Cannot open state database: %v", err)
	}
	defer writeDB.Close()
	defer readDB.Close()

	if err := loadConfig(); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}
	log.Printf("Loaded state from %s", stateFile)

	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGHUP, syscall.SIGQUIT)
	go handleSignal(chSig)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleFunc)
	mux.HandleFunc("/ip", handleIP)
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "User-Agent: *\nDisallow: /", http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", listenHost, listenPort), mux))
}
