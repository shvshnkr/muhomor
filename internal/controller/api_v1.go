package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/profiles"
)

func (d *Daemon) registerV1(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/settings", d.handleSettingsGet)
	mux.HandleFunc("PUT /v1/settings", d.handleSettingsPut)
	mux.HandleFunc("GET /v1/profiles", d.handleProfilesList)
	mux.HandleFunc("POST /v1/profiles/import", d.handleProfilesImport)
	mux.HandleFunc("PUT /v1/profiles/{id}/enabled", d.handleProfileEnabled)
	mux.HandleFunc("POST /v1/profiles/{id}/connect", d.handleProfileConnect)
	mux.HandleFunc("GET /v1/events", d.handleEvents)
	mux.HandleFunc("POST /v1/service/ping", d.handleServicePing)
	mux.HandleFunc("POST /v1/service/bulk-ping-all", d.handleServiceBulkPingAll)
	mux.HandleFunc("POST /v1/daemon/shutdown", d.handleDaemonShutdown)
	d.registerGroupsV1(mux)
}

func (d *Daemon) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	set, err := d.Runtime.Store.LoadSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	qp, _ := d.Runtime.Store.RouteQuickProfile(r.Context())
	set.RouteQuickProfile = qp
	writeJSON(w, http.StatusOK, api.SettingsFromStore(set))
}

func (d *Daemon) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var req api.Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	set := req.ToStore()
	if err := d.Runtime.Store.SaveSettings(r.Context(), set); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if req.RouteQuickProfile >= 0 {
		_ = d.Runtime.Store.SetRouteQuickProfile(r.Context(), req.RouteQuickProfile)
	}
	d.Runtime.refreshBuildOptions(r.Context())
	d.Runtime.publishStatusEvent("settings")
	writeJSON(w, http.StatusOK, api.SettingsFromStore(set))
}

func (d *Daemon) handleProfilesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list, err := d.Runtime.Store.ListAllProfiles(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	groups, _ := d.Runtime.Store.ListGroups(ctx)
	gname := map[int64]string{}
	for _, g := range groups {
		gname[g.ID] = g.Name
	}
	out := make([]api.Profile, len(list))
	for i, p := range list {
		out[i] = api.ProfileFromStore(p)
		out[i].GroupName = gname[p.GroupID]
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": out})
}

func (d *Daemon) handleProfilesImport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var lines []string
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file field required"})
			return
		}
		defer f.Close()
		res, err := profiles.ImportReader(ctx, d.Runtime.Store, f)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": toAPIImportResults(res)})
		return
	}
	var req api.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.URI != "" {
		lines = []string{req.URI}
	} else {
		lines = req.Lines
	}
	if len(lines) == 0 {
		body, _ := io.ReadAll(r.Body)
		if len(strings.TrimSpace(string(body))) > 0 {
			lines = strings.Split(string(body), "\n")
		}
	}
	var res []profiles.ImportResult
	var err error
	if req.GroupID > 0 {
		res, err = profiles.ImportLinesToGroup(ctx, d.Runtime.Store, req.GroupID, lines)
	} else {
		res, err = profiles.ImportLines(ctx, d.Runtime.Store, lines)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": toAPIImportResults(res)})
}

func toAPIImportResults(res []profiles.ImportResult) []api.ImportResult {
	out := make([]api.ImportResult, len(res))
	for i, r := range res {
		out[i] = api.ImportResult{ID: r.ID, Name: r.Name, Type: r.Type, Skip: r.Skip, Reason: r.Reason}
	}
	return out
}

func (d *Daemon) handleProfileEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/enabled")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req api.ProfileEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := d.Runtime.Store.SetProfileEnabled(r.Context(), id, req.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "enabled": req.Enabled})
}

func (d *Daemon) handleProfileConnect(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r.URL.Path, "/connect")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := d.Runtime.ConnectProfile(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	_ = d.Runtime.WriteStatusFile()
	st := d.Runtime.statusSnapshot(r.Context())
	writeJSON(w, http.StatusOK, st)
}

func (d *Daemon) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := d.Runtime.Events.Subscribe()
	defer d.Runtime.Events.Unsubscribe(ch)

	// initial snapshot
	st := d.Runtime.statusSnapshot(r.Context())
	initial, _ := json.Marshal(api.Event{Type: "status", Timestamp: time.Now().UnixMilli(), Status: &st})
	fmt.Fprintf(w, "data: %s\n\n", initial)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (d *Daemon) handleServicePing(w http.ResponseWriter, r *http.Request) {
	resp, err := d.Runtime.Ping(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (d *Daemon) handleServiceBulkPingAll(w http.ResponseWriter, r *http.Request) {
	resp, err := d.Runtime.PingBulkAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func pathID(path, suffix string) (int64, error) {
	// /v1/profiles/123/enabled
	path = strings.TrimPrefix(path, "/v1/profiles/")
	path = strings.TrimSuffix(path, suffix)
	path = strings.Trim(path, "/")
	return strconv.ParseInt(path, 10, 64)
}
