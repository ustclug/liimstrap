package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

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
	if time.Since(ci.Time) > aliveTimeout {
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
	configFile   string
	listenPort   int
	dumpTemplate bool
	stateFile    string

	aliveTimeout time.Duration
	clientData   []ClientInfo
	clientIndex  map[string]int
	clientLock   sync.Mutex

	//go:embed index.html
	indexTemplateStr string
	indexTemplate    template.Template = *template.Must(template.New("index").Parse(indexTemplateStr))
)

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
	clientLock.Lock()
	defer clientLock.Unlock()

	newData := make([]ClientInfo, len(config.Machines))
	newIndex := make(map[string]int, len(config.Machines)+1)
	for i, m := range config.Machines {
		m.Mac = NormalizeMac(m.Mac)
		if i, ok := clientIndex[m.Mac]; ok {
			newData[i] = clientData[i]
		}
		newData[i].Name = m.Name
		newData[i].Mac = m.Mac
		newIndex[m.Mac] = i
	}
	if _, ok := newIndex[""]; !ok {
		newIndex[""] = len(newData)
		newData = append(newData, ClientInfo{Name: "Unknown"})
	}
	aliveTimeout = time.Duration(config.Timeout) * time.Second
	clientData = newData
	clientIndex = newIndex
	log.Printf("Loaded configuration, total %d clients", len(clientData))
	return nil
}

func saveState() error {
	f, err := os.Create(stateFile)
	if err != nil {
		return err
	}
	defer f.Close()
	clientLock.Lock()
	defer clientLock.Unlock()
	return json.NewEncoder(f).Encode(clientData)
}

func loadState() error {
	f, err := os.Open(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("State file %s not found, skipping\n", stateFile)
			return nil
		}
		return err
	}
	defer f.Close()
	var newData []ClientInfo
	if err := json.NewDecoder(f).Decode(&newData); err != nil {
		return err
	}
	clientLock.Lock()
	defer clientLock.Unlock()
	for _, info := range newData {
		info.Mac = NormalizeMac(info.Mac)
		if index, ok := clientIndex[info.Mac]; ok {
			clientData[index] = info
		} else {
			clientData = append(clientData, info)
			clientIndex[info.Mac] = len(clientData) - 1
		}
	}
	return nil
}

func handleSignal(chSig <-chan os.Signal) {
	for sig := range chSig {
		switch sig {
		case syscall.SIGHUP:
			log.Printf("Received SIGHUP\n")
			if err := loadConfig(); err != nil {
				log.Printf("Cannot reload config: %v", err)
			}
			if err := loadState(); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("Cannot load state: %v", err)
			}
		case syscall.SIGQUIT:
			log.Printf("Received SIGQUIT\n")
			err := saveState()
			if err != nil {
				log.Printf("Cannot save state: %v", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render HTML list
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusOK)

		// Construct data
		clientLock.Lock()
		defer clientLock.Unlock()
		err := indexTemplate.Execute(w, clientData)
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

		clientLock.Lock()
		defer clientLock.Unlock()
		i, ok := clientIndex[mac]
		if !ok {
			i, ok = clientIndex[""]
			if !ok {
				http.Error(w, "OK", http.StatusOK)
				return
			}
		}
		d := &clientData[i]

		ip := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
		if ip[0] == '[' {
			ip = ip[1 : len(ip)-1]
		}
		d.IP = ip
		d.Time = time.Now()
		d.Version = version
		d.Uptime = time.Duration(uptime) * time.Second
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
	}

	clientLock.Lock()
	defer clientLock.Unlock()
	for _, d := range clientData {
		if d.IP == q {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(d)
			return
		}
	}
	http.Error(w, fmt.Sprintf("IP %s not found", q), http.StatusNotFound)
}

func main() {
	flag.StringVar(&configFile, "c", "clients.yaml", "YAML config of clients")
	flag.IntVar(&listenPort, "p", 3000, "port to listen on")
	flag.StringVar(&stateFile, "s", "/var/lib/liims-monitor/state.json", "save state file")
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

	if err := loadConfig(); err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}
	if err := loadState(); err != nil {
		log.Printf("Cannot load saved state: %v", err)
	} else {
		log.Printf("Loaded state from %s", stateFile)
	}

	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGHUP, syscall.SIGQUIT)
	go handleSignal(chSig)

	go func() {
		for range time.NewTicker(30 * time.Second).C {
			saveState()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleFunc)
	mux.HandleFunc("/ip", handleIP)
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "User-Agent: *\nDisallow: /", http.StatusOK)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", listenPort), mux))
}
