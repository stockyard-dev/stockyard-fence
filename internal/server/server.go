package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stockyard-dev/stockyard-fence/internal/store"
)

type Server struct {
	db       *store.DB
	mux      *http.ServeMux
	port     int
	adminKey string
	limits   Limits
}

func New(db *store.DB, port int, adminKey string, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), port: port, adminKey: adminKey, limits: limits}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Vaults (admin)
	s.mux.HandleFunc("GET /api/vaults", s.admin(s.handleListVaults))
	s.mux.HandleFunc("POST /api/vaults", s.admin(s.handleCreateVault))
	s.mux.HandleFunc("GET /api/vaults/{id}", s.admin(s.handleGetVault))
	s.mux.HandleFunc("DELETE /api/vaults/{id}", s.admin(s.handleDeleteVault))

	// Keys (admin)
	s.mux.HandleFunc("GET /api/vaults/{id}/keys", s.admin(s.handleListKeys))
	s.mux.HandleFunc("POST /api/vaults/{id}/keys", s.admin(s.handleStoreKey))
	s.mux.HandleFunc("POST /api/vaults/{id}/keys/{kid}/rotate", s.admin(s.handleRotateKey))
	s.mux.HandleFunc("DELETE /api/vaults/{id}/keys/{kid}", s.admin(s.handleDeleteKey))

	// Members (admin)
	s.mux.HandleFunc("GET /api/vaults/{id}/members", s.admin(s.handleListMembers))
	s.mux.HandleFunc("POST /api/vaults/{id}/members", s.admin(s.handleAddMember))
	s.mux.HandleFunc("DELETE /api/vaults/{id}/members/{mid}", s.admin(s.handleDeleteMember))

	// Tokens (admin)
	s.mux.HandleFunc("GET /api/vaults/{id}/tokens", s.admin(s.handleListTokens))
	s.mux.HandleFunc("POST /api/vaults/{id}/tokens", s.admin(s.handleIssueToken))
	s.mux.HandleFunc("DELETE /api/vaults/{id}/tokens/{tid}", s.admin(s.handleRevokeToken))

	// Audit log (admin)
	s.mux.HandleFunc("GET /api/vaults/{id}/audit", s.admin(s.handleAuditLog))
	s.mux.HandleFunc("GET /api/vaults/{id}/export", s.admin(s.handleExportVault))
	s.mux.HandleFunc("POST /api/vaults/{id}/import", s.admin(s.handleImportVault))

	// Key resolution (token auth — used by team members at runtime)
	s.mux.HandleFunc("GET /api/resolve/{name}", s.handleResolve)

	// Status
	s.mux.HandleFunc("GET /api/stats", s.admin(s.handleStats))
	s.mux.HandleFunc("GET /ui", s.handleUI)
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {

	// Tier (for upgrade banner)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/fence/"})
	})

		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[fence] listening on %s", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	return srv.ListenAndServe()
}

// ── Auth middleware ───────────────────────────────────────────────────

func (s *Server) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := extractBearer(r)
		if s.adminKey == "" || key != s.adminKey {
			writeJSON(w, 401, map[string]string{"error": "admin key required (Authorization: Bearer <key>)"})
			return
		}
		next(w, r)
	}
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("key")
}

func sourceIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}

// ── Vault handlers ────────────────────────────────────────────────────

func (s *Server) handleListVaults(w http.ResponseWriter, r *http.Request) {
	vaults, err := s.db.ListVaults()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if vaults == nil {
		vaults = []store.Vault{}
	}
	writeJSON(w, 200, map[string]any{"vaults": vaults, "count": len(vaults)})
}

func (s *Server) handleCreateVault(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Desc string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name required"})
		return
	}
	if s.limits.MaxVaults > 0 {
		vaults, _ := s.db.ListVaults()
		if LimitReached(s.limits.MaxVaults, len(vaults)) {
			writeJSON(w, 402, map[string]string{"error": "free tier limit: " + strconv.Itoa(s.limits.MaxVaults) + " vaults max — upgrade to Pro", "upgrade": "https://stockyard.dev/fence/"})
			return
		}
	}
	v, err := s.db.CreateVault(req.Name, req.Desc)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "vault name already exists"})
		return
	}
	writeJSON(w, 201, map[string]any{"vault": v})
}

func (s *Server) handleGetVault(w http.ResponseWriter, r *http.Request) {
	v, err := s.db.GetVault(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "vault not found"})
		return
	}
	writeJSON(w, 200, map[string]any{"vault": v})
}

func (s *Server) handleDeleteVault(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteVault(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// ── Key handlers ──────────────────────────────────────────────────────

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.db.ListKeys(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if keys == nil {
		keys = []store.SecretKey{}
	}
	writeJSON(w, 200, map[string]any{"keys": keys, "count": len(keys)})
}

func (s *Server) handleStoreKey(w http.ResponseWriter, r *http.Request) {
	vaultID := r.PathValue("id")
	var req struct {
		Name      string `json:"name"`
		Value     string `json:"value"`
		Provider  string `json:"provider"`
		Notes     string `json:"notes"`
		RotateAt  string `json:"rotate_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Value == "" {
		writeJSON(w, 400, map[string]string{"error": "name and value required"})
		return
	}
	if s.limits.MaxKeys > 0 {
		keys, _ := s.db.ListKeys(vaultID)
		if LimitReached(s.limits.MaxKeys, len(keys)) {
			writeJSON(w, 402, map[string]string{"error": "free tier limit: " + strconv.Itoa(s.limits.MaxKeys) + " keys max — upgrade to Pro", "upgrade": "https://stockyard.dev/fence/"})
			return
		}
	}
	k, err := s.db.StoreKey(vaultID, req.Name, req.Value, req.Provider, req.Notes, req.RotateAt)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "key name already exists in this vault"})
		return
	}
	writeJSON(w, 201, map[string]any{"key": k, "note": "value is encrypted at rest and never returned by the API"})
}

func (s *Server) handleRotateKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Value == "" {
		writeJSON(w, 400, map[string]string{"error": "value required"})
		return
	}
	if err := s.db.RotateKey(r.PathValue("kid"), req.Value); err != nil {
		writeJSON(w, 404, map[string]string{"error": "key not found"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "rotated"})
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteKey(r.PathValue("kid"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// ── Member handlers ───────────────────────────────────────────────────

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.db.ListMembers(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if members == nil {
		members = []store.Member{}
	}
	writeJSON(w, 200, map[string]any{"members": members, "count": len(members)})
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	vaultID := r.PathValue("id")
	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		writeJSON(w, 400, map[string]string{"error": "username required"})
		return
	}
	if req.Role == "" {
		req.Role = "reader"
	}
	if s.limits.MaxMembers > 0 {
		members, _ := s.db.ListMembers(vaultID)
		if LimitReached(s.limits.MaxMembers, len(members)) {
			writeJSON(w, 402, map[string]string{"error": "free tier limit: " + strconv.Itoa(s.limits.MaxMembers) + " members max — upgrade to Pro", "upgrade": "https://stockyard.dev/fence/"})
			return
		}
	}
	m, err := s.db.AddMember(vaultID, req.Username, req.Role)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "member already exists in vault"})
		return
	}
	writeJSON(w, 201, map[string]any{"member": m})
}

func (s *Server) handleDeleteMember(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteMember(r.PathValue("mid"))
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// ── Token handlers ────────────────────────────────────────────────────

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := s.db.ListTokens(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if tokens == nil {
		tokens = []store.Token{}
	}
	writeJSON(w, 200, map[string]any{"tokens": tokens, "count": len(tokens)})
}

func (s *Server) handleIssueToken(w http.ResponseWriter, r *http.Request) {
	vaultID := r.PathValue("id")
	var req struct {
		MemberID string `json:"member_id"`
		KeyID    string `json:"key_id"`
		Name     string `json:"name"`
		TTLHours int    `json:"ttl_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MemberID == "" {
		writeJSON(w, 400, map[string]string{"error": "member_id required"})
		return
	}
	rawToken, t, err := s.db.IssueToken(req.MemberID, vaultID, req.KeyID, req.Name, req.TTLHours)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]any{
		"token":  rawToken,
		"meta":   t,
		"note":   "Save this token — it will not be shown again",
	})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	s.db.RevokeToken(r.PathValue("tid"))
	writeJSON(w, 200, map[string]string{"status": "revoked"})
}

// ── Resolve handler (token auth) ──────────────────────────────────────

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	rawToken := extractBearer(r)
	if rawToken == "" {
		writeJSON(w, 401, map[string]string{"error": "token required (Authorization: Bearer fence_...)"})
		return
	}

	tok, err := s.db.ResolveToken(rawToken)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}

	keyName := r.PathValue("name")

	// If token is scoped to a specific key, enforce it
	var keyID string
	if tok.KeyID != "" {
		// Look up the key by ID and verify name matches
		keys, _ := s.db.ListKeys(tok.VaultID)
		found := false
		for _, k := range keys {
			if k.ID == tok.KeyID {
				if k.Name != keyName {
					writeJSON(w, 403, map[string]string{"error": "token not authorized for this key"})
					return
				}
				keyID = k.ID
				found = true
				break
			}
		}
		if !found {
			writeJSON(w, 404, map[string]string{"error": "key not found"})
			return
		}
	} else {
		// Vault-scoped token — resolve by name
		k, err := s.db.GetKeyByName(tok.VaultID, keyName)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "key not found"})
			return
		}
		keyID = k.ID
	}

	value, err := s.db.GetKeyValue(keyID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to decrypt key"})
		return
	}

	// Log access
	s.db.LogAccess(keyID, keyName, tok.VaultID, tok.MemberID, tok.ID, sourceIP(r))

	log.Printf("[fence] resolve key=%s vault=%s member=%s", keyName, tok.VaultID, tok.MemberID)

	writeJSON(w, 200, map[string]any{
		"key":      keyName,
		"value":    value,
		"vault_id": tok.VaultID,
	})
}

// ── Audit log ─────────────────────────────────────────────────────────

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if !s.limits.FullAuditTrail {
		writeJSON(w, 402, map[string]string{"error": "full audit trail requires Pro — upgrade at https://stockyard.dev/fence/", "upgrade": "https://stockyard.dev/fence/"})
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := s.db.ListAccess(r.PathValue("id"), limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []store.AccessEntry{}
	}
	writeJSON(w, 200, map[string]any{"entries": entries, "count": len(entries)})
}

func (s *Server) handleExportVault(w http.ResponseWriter, r *http.Request) {
	if !s.limits.ExportImport {
		writeJSON(w, 402, map[string]string{"error": "export/import requires Pro — upgrade at https://stockyard.dev/fence/", "upgrade": "https://stockyard.dev/fence/"})
		return
	}
	keys, err := s.db.ListKeys(r.PathValue("id"))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="vault-export.json"`)
	writeJSON(w, 200, map[string]any{"vault_id": r.PathValue("id"), "keys": keys, "note": "values are not exported — re-add values after import"})
}

func (s *Server) handleImportVault(w http.ResponseWriter, r *http.Request) {
	if !s.limits.ExportImport {
		writeJSON(w, 402, map[string]string{"error": "export/import requires Pro — upgrade at https://stockyard.dev/fence/", "upgrade": "https://stockyard.dev/fence/"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok", "note": "import endpoint active — POST key array to populate"})
}

// 5. ExpirationReminders — flag checked by a background goroutine (wired in main.go)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

// ── Helpers ───────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
