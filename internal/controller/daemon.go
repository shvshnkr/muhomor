package controller

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Daemon serves ctl over Unix socket (Linux) or TCP fallback.
type Daemon struct {
	Runtime *Runtime
	Log     *slog.Logger
	ln      net.Listener
	wg      sync.WaitGroup
}

func (d *Daemon) ListenAndServe(ctx context.Context, socketPath string) error {
	_ = os.Remove(socketPath)
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return err
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:8751")
		if err != nil {
			return err
		}
	}
	d.ln = ln
	if d.Log == nil {
		d.Log = slog.Default()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/service/start", d.handleStart)
	mux.HandleFunc("/v1/service/stop", d.handleStop)
	mux.HandleFunc("/v1/service/reload", d.handleReload)
	mux.HandleFunc("/v1/service/status", d.handleStatus)
	mux.HandleFunc("/v1/simple/connect", d.handleStart)
	mux.HandleFunc("/v1/simple/adapt", d.handleAdapt)
	mux.HandleFunc("/v1/service/chain", d.handleChain)
	mux.HandleFunc("/v1/logs/export", d.handleExportLog)
	mux.HandleFunc("/v1/update/check", d.handleUpdateCheck)
	mux.HandleFunc("/v1/update/install", d.handleUpdateInstall)
	srv := &http.Server{Handler: mux}
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		<-ctx.Done()
		_ = srv.Close()
	}()
	d.Log.Info("daemon listening", "addr", ln.Addr().String())
	err = srv.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (d *Daemon) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if err := d.Runtime.Start(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = d.Runtime.WriteStatusFile()
	writeJSON(w, http.StatusOK, map[string]string{"state": string(d.Runtime.Status().State)})
}

func (d *Daemon) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	_ = d.Runtime.Stop(r.Context())
	_ = d.Runtime.WriteStatusFile()
	writeJSON(w, http.StatusOK, map[string]string{"state": string(d.Runtime.Status().State)})
}

func (d *Daemon) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if err := d.Runtime.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = d.Runtime.WriteStatusFile()
	writeJSON(w, http.StatusOK, map[string]string{"state": string(d.Runtime.Status().State)})
}

func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	_ = d.Runtime.WriteStatusFile()
	writeJSON(w, http.StatusOK, d.Runtime.Status())
}

func (d *Daemon) handleChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	ids := parseChainIDs(string(body))
	if len(ids) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "profile ids required: {\"ids\":[1,2]}"})
		return
	}
	if err := d.Runtime.StartChain(r.Context(), ids); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": d.Runtime.Status().State, "ids": ids})
}

func parseChainIDs(body string) []int64 {
	body = strings.TrimSpace(body)
	if strings.HasPrefix(body, "{") {
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if json.Unmarshal([]byte(body), &req) == nil {
			return req.IDs
		}
	}
	var out []int64
	for _, p := range strings.Split(body, ",") {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func (d *Daemon) handleAdapt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	d.Runtime.Adapt(r.Context(), "api")
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (d *Daemon) handleExportLog(w http.ResponseWriter, r *http.Request) {
	path, err := d.Runtime.ExportSimpleLog()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (d *Daemon) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"result":  "unsupported",
		"note":    "app self-update not implemented; use package manager",
		"event":   "update-check",
	})
}

func (d *Daemon) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"result": "unsupported", "note": "Phase 3: install via distro package"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
