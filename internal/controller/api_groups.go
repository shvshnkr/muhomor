package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/api"
	"github.com/muhomor/muhomor/internal/profiles"
	"github.com/muhomor/muhomor/internal/store"
	"github.com/muhomor/muhomor/internal/subscription"
)

func (d *Daemon) registerGroupsV1(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/groups", d.handleGroupsList)
	mux.HandleFunc("POST /v1/groups", d.handleGroupsCreate)
	mux.HandleFunc("PUT /v1/groups/{id}", d.handleGroupsUpdate)
	mux.HandleFunc("DELETE /v1/groups/{id}", d.handleGroupsDelete)
	mux.HandleFunc("POST /v1/groups/{id}/import", d.handleGroupsImport)
	mux.HandleFunc("POST /v1/groups/{id}/test-all", d.handleGroupsTestAll)
	mux.HandleFunc("POST /v1/groups/{id}/refresh", d.handleGroupsRefresh)
	mux.HandleFunc("POST /v1/groups/{id}/servers", d.handleGroupsAddServer)
	mux.HandleFunc("DELETE /v1/profiles/{id}", d.handleProfileDelete)
	mux.HandleFunc("POST /v1/profiles/{id}/delay-test", d.handleProfileDelayTest)
}

func (d *Daemon) groupUserAgent(ctx context.Context, id int64) string {
	v, _ := d.Runtime.Store.GetKV(ctx, store.KeyGroupUserAgent(id))
	return v
}

func (d *Daemon) handleGroupsList(w http.ResponseWriter, r *http.Request) {
	groups, err := d.Runtime.Store.ListGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	counts, _ := d.Runtime.Store.ProfileCountByGroup(r.Context())
	out := make([]api.Group, len(groups))
	for i, g := range groups {
		builtin := g.Name == store.BuiltinWLGroupName
		out[i] = api.GroupFromStore(g, counts[g.ID], builtin, d.groupUserAgent(r.Context(), g.ID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out})
}

func (d *Daemon) handleGroupsCreate(w http.ResponseWriter, r *http.Request) {
	var req api.GroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	kind := req.Kind
	if kind == "" {
		if strings.TrimSpace(req.SubscriptionLink) != "" {
			kind = store.GroupKindSubscription
		} else {
			kind = store.GroupKindManual
		}
	}
	if kind != store.GroupKindSubscription && kind != store.GroupKindManual {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind must be subscription or manual"})
		return
	}
	id, err := d.Runtime.Store.CreateGroup(r.Context(), req.Name, req.SubscriptionLink, kind)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	g, _ := d.Runtime.Store.ListGroups(r.Context())
	for _, x := range g {
		if x.ID == id {
			writeJSON(w, http.StatusOK, api.GroupFromStore(x, 0, false, d.groupUserAgent(r.Context(), id)))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (d *Daemon) handleGroupsUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req api.GroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Kind != "" {
		_ = d.Runtime.Store.SetGroupKind(r.Context(), id, req.Kind)
	}
	if req.SubscriptionLink != "" {
		_ = d.Runtime.Store.SetGroupSubscription(r.Context(), id, req.SubscriptionLink)
		_ = d.Runtime.Store.SetKV(r.Context(), store.KeyGroupUserAgent(id), "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "ok": true})
}

func (d *Daemon) handleGroupsRefresh(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(strings.TrimSuffix(r.URL.Path, "/refresh"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.Runtime.setActivity(r.Context(), "Обновление подписки…")
	added, err := d.Runtime.RefreshSubscriptionGroup(r.Context(), id)
	ua := d.groupUserAgent(r.Context(), id)
	if err != nil {
		if d.Log != nil {
			d.Log.Warn("subscription refresh", "group_id", id, "err", err)
		}
		d.Runtime.clearActivity(r.Context())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	uaMode := subscription.UAModeLabel(ua)
	if d.Log != nil {
		d.Log.Info("subscription refresh ok", "group_id", id, "imported", added, "ua_mode", uaMode)
	}
	d.Runtime.clearActivity(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"group_id": id, "imported": added, "user_agent_mode": uaMode,
	})
}

func (d *Daemon) handleGroupsAddServer(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(strings.TrimSuffix(r.URL.Path, "/servers"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	g, err := d.Runtime.Store.ListGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var grp *store.Group
	for i := range g {
		if g[i].ID == id {
			grp = &g[i]
			break
		}
	}
	if grp == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "group not found"})
		return
	}
	if grp.Kind != store.GroupKindManual {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only manual groups accept individual servers"})
		return
	}
	var req api.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.URI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uri required"})
		return
	}
	res, err := profiles.ImportLinesToGroup(r.Context(), d.Runtime.Store, id, []string{req.URI})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": toAPIImportResults(res)})
}

func (d *Daemon) handleGroupsDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := d.Runtime.Store.DeleteGroup(r.Context(), id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (d *Daemon) handleGroupsImport(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(strings.TrimSuffix(r.URL.Path, "/import"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req api.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	lines := req.Lines
	if req.URI != "" {
		lines = []string{req.URI}
	}
	// Import URI lines into group; subscription groups may paste vless lines; use /refresh for HTTP sub.
	res, err := profiles.ImportLinesToGroup(r.Context(), d.Runtime.Store, id, lines)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": toAPIImportResults(res)})
}

func (d *Daemon) handleGroupsTestAll(w http.ResponseWriter, r *http.Request) {
	id, err := pathGroupID(strings.TrimSuffix(r.URL.Path, "/test-all"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.Runtime.setActivity(r.Context(), "Тест списка…")
	tested, ok, err := d.Runtime.TestGroupDelays(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	d.Runtime.clearActivity(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"tested": tested, "ok": ok})
}

func (d *Daemon) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathProfileID(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := d.Runtime.Store.DeleteProfile(r.Context(), id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (d *Daemon) handleProfileDelayTest(w http.ResponseWriter, r *http.Request) {
	id, err := pathProfileID(strings.TrimSuffix(r.URL.Path, "/delay-test"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	res, err := d.Runtime.TestProfileDelay(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func pathGroupID(path string) (int64, error) {
	path = strings.TrimPrefix(path, "/v1/groups/")
	path = strings.Trim(path, "/")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[:i]
	}
	return strconv.ParseInt(path, 10, 64)
}

func pathProfileID(path string) (int64, error) {
	path = strings.TrimPrefix(path, "/v1/profiles/")
	path = strings.Trim(path, "/")
	return strconv.ParseInt(path, 10, 64)
}
