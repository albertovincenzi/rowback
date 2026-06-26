package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// job holds the state of the single active/last extraction for the UI.
type job struct {
	mu      sync.Mutex
	running bool
	last    Progress
	err     string
	outPath string
	result  *Result
	subs    map[chan Progress]struct{}
}

func newJob() *job { return &job{subs: map[chan Progress]struct{}{}} }

func (j *job) publish(p Progress) {
	j.mu.Lock()
	j.last = p
	for ch := range j.subs {
		select {
		case ch <- p:
		default:
		}
	}
	j.mu.Unlock()
}

func (j *job) subscribe() chan Progress {
	ch := make(chan Progress, 8)
	j.mu.Lock()
	j.subs[ch] = struct{}{}
	j.mu.Unlock()
	return ch
}

func (j *job) unsubscribe(ch chan Progress) {
	j.mu.Lock()
	delete(j.subs, ch)
	j.mu.Unlock()
	close(ch)
}

func serveUI(addr, defaultDump string) error {
	j := newJob()

	mux := http.NewServeMux()

	// Serve the embedded production frontend (index.html, app.css, app.js).
	mux.Handle("/", http.FileServerFS(webRoot()))

	// /api/config feeds the client its defaults as JSON, so the HTML is served
	// verbatim from the embedded file rather than string-templated in Go.
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"defaultDump":  defaultDump,
			"defaultQuery": defaultQuery,
		})
	})

	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var cfg Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if cfg.OutPath == "" {
			cfg.OutPath = "restore_reservations.sql"
		}
		if cfg.Format == "" {
			cfg.Format = "insert"
		}

		j.mu.Lock()
		if j.running {
			j.mu.Unlock()
			http.Error(w, "a job is already running", http.StatusConflict)
			return
		}
		// Validate up front so the UI gets an immediate, friendly error.
		if fi, err := os.Stat(cfg.DumpPath); err != nil || fi.IsDir() {
			j.mu.Unlock()
			http.Error(w, "dump file not found: "+cfg.DumpPath, http.StatusBadRequest)
			return
		}
		// A relative output path would land in the process cwd, which is "/"
		// (read-only) when launched from an .app bundle. Resolve it to a
		// writable directory next to the dump (or the home dir).
		cfg.OutPath = resolveOutPath(cfg.OutPath, cfg.DumpPath)
		if dir := filepath.Dir(cfg.OutPath); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				j.mu.Unlock()
				http.Error(w, "cannot create restore folder: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
		if _, err := ParseQuery(cfg.Query); err != nil {
			j.mu.Unlock()
			http.Error(w, "query: "+err.Error(), http.StatusBadRequest)
			return
		}
		j.running = true
		j.err = ""
		j.outPath = cfg.OutPath
		j.result = nil
		j.mu.Unlock()

		go func() {
			res := newResult()
			_, err := Extract(cfg, j.publish, res)
			j.mu.Lock()
			j.running = false
			if err != nil {
				j.err = err.Error()
			} else {
				j.result = res
			}
			j.mu.Unlock()
			j.publish(Progress{Done: true, Phase: "done", Message: errMsg(err)})
		}()

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"started"}`)
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := j.subscribe()
		defer j.unsubscribe(ch)

		// send current snapshot immediately
		j.mu.Lock()
		first := j.last
		errNow := j.err
		j.mu.Unlock()
		writeSSE(w, first, errNow)
		fl.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case p := <-ch:
				j.mu.Lock()
				errNow := j.err
				j.mu.Unlock()
				writeSSE(w, p, errNow)
				fl.Flush()
			}
		}
	})

	mux.HandleFunc("/api/result", func(w http.ResponseWriter, r *http.Request) {
		j.mu.Lock()
		res := j.result
		j.mu.Unlock()
		if res == nil {
			http.Error(w, "no result yet", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		j.mu.Lock()
		path := j.outPath
		j.mu.Unlock()
		if path == "" {
			http.Error(w, "no output yet", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=\"restore_reservations.sql\"")
		http.ServeFile(w, r, path)
	})

	fmt.Printf("rowback UI → http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func writeSSE(w http.ResponseWriter, p Progress, errStr string) {
	type payload struct {
		Progress
		Err string  `json:"err"`
		Pct float64 `json:"pct"`
	}
	b, _ := json.Marshal(payload{Progress: p, Err: errStr, Pct: pct(p.BytesRead, p.TotalBytes)})
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func errMsg(err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return ""
}

// resolveOutPath turns a relative output path into an absolute one in a writable
// directory: prefer the dump's directory, then $HOME, then the OS temp dir. This
// avoids "read-only file system" when the server runs from an .app bundle (cwd=/).
func resolveOutPath(out, dumpPath string) string {
	if filepath.IsAbs(out) {
		return out
	}
	var candidates []string
	if d := filepath.Dir(dumpPath); d != "" && d != "." {
		candidates = append(candidates, d)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home)
	}
	candidates = append(candidates, os.TempDir())

	for _, dir := range candidates {
		if isWritableDir(dir) {
			return filepath.Join(dir, out)
		}
	}
	return filepath.Join(os.TempDir(), out)
}

func isWritableDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".pr-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}
