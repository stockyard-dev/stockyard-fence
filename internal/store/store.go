package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn      *sql.DB
	encKey    []byte // 32-byte AES-256 key
}

func Open(dataDir string, encryptionKey []byte) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(encryptionKey))
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "fence.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(1) // single writer for SQLite
	db := &DB{conn: conn, encKey: encryptionKey}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS vaults (
    id   TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    desc TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS secret_keys (
    id         TEXT PRIMARY KEY,
    vault_id   TEXT NOT NULL REFERENCES vaults(id),
    name       TEXT NOT NULL,
    ciphertext TEXT NOT NULL,
    provider   TEXT DEFAULT '',
    notes      TEXT DEFAULT '',
    rotate_at  TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(vault_id, name)
);
CREATE INDEX IF NOT EXISTS idx_keys_vault ON secret_keys(vault_id);

CREATE TABLE IF NOT EXISTS members (
    id         TEXT PRIMARY KEY,
    vault_id   TEXT NOT NULL REFERENCES vaults(id),
    username   TEXT NOT NULL,
    role       TEXT DEFAULT 'reader',
    enabled    INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(vault_id, username)
);
CREATE INDEX IF NOT EXISTS idx_members_vault ON members(vault_id);

CREATE TABLE IF NOT EXISTS tokens (
    id         TEXT PRIMARY KEY,
    member_id  TEXT NOT NULL REFERENCES members(id),
    vault_id   TEXT NOT NULL,
    key_id     TEXT DEFAULT '',
    name       TEXT DEFAULT '',
    token_hash TEXT UNIQUE NOT NULL,
    expires_at TEXT DEFAULT '',
    enabled    INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_tokens_member ON tokens(member_id);

CREATE TABLE IF NOT EXISTS access_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    key_id     TEXT NOT NULL,
    key_name   TEXT NOT NULL,
    vault_id   TEXT NOT NULL,
    member_id  TEXT NOT NULL,
    token_id   TEXT NOT NULL,
    source_ip  TEXT DEFAULT '',
    accessed_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_access_key ON access_log(key_id);
CREATE INDEX IF NOT EXISTS idx_access_time ON access_log(accessed_at);
`)
	return err
}

// ── Encryption ────────────────────────────────────────────────────────

func (db *DB) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(db.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (db *DB) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(db.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphered := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphered, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// ── Vault ─────────────────────────────────────────────────────────────

type Vault struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Desc      string `json:"description"`
	KeyCount  int    `json:"key_count"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) CreateVault(name, desc string) (*Vault, error) {
	id := "vlt_" + genID(8)
	_, err := db.conn.Exec("INSERT INTO vaults (id,name,desc) VALUES (?,?,?)", id, name, desc)
	if err != nil {
		return nil, err
	}
	return &Vault{ID: id, Name: name, Desc: desc, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (db *DB) ListVaults() ([]Vault, error) {
	rows, err := db.conn.Query(`
		SELECT v.id, v.name, v.desc, v.created_at,
		       (SELECT COUNT(*) FROM secret_keys WHERE vault_id=v.id)
		FROM vaults v ORDER BY v.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Vault
	for rows.Next() {
		var v Vault
		rows.Scan(&v.ID, &v.Name, &v.Desc, &v.CreatedAt, &v.KeyCount)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (db *DB) GetVault(id string) (*Vault, error) {
	var v Vault
	err := db.conn.QueryRow(`
		SELECT v.id, v.name, v.desc, v.created_at,
		       (SELECT COUNT(*) FROM secret_keys WHERE vault_id=v.id)
		FROM vaults v WHERE v.id=?`, id).
		Scan(&v.ID, &v.Name, &v.Desc, &v.CreatedAt, &v.KeyCount)
	return &v, err
}

func (db *DB) DeleteVault(id string) error {
	db.conn.Exec("DELETE FROM secret_keys WHERE vault_id=?", id)
	db.conn.Exec("DELETE FROM members WHERE vault_id=?", id)
	db.conn.Exec("DELETE FROM tokens WHERE vault_id=?", id)
	_, err := db.conn.Exec("DELETE FROM vaults WHERE id=?", id)
	return err
}

// ── Secret Keys ───────────────────────────────────────────────────────

type SecretKey struct {
	ID        string `json:"id"`
	VaultID   string `json:"vault_id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Notes     string `json:"notes"`
	RotateAt  string `json:"rotate_at,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (db *DB) StoreKey(vaultID, name, value, provider, notes, rotateAt string) (*SecretKey, error) {
	ciphertext, err := db.encrypt(value)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	id := "key_" + genID(10)
	_, err = db.conn.Exec(
		`INSERT INTO secret_keys (id,vault_id,name,ciphertext,provider,notes,rotate_at) VALUES (?,?,?,?,?,?,?)`,
		id, vaultID, name, ciphertext, provider, notes, rotateAt)
	if err != nil {
		return nil, err
	}
	return &SecretKey{ID: id, VaultID: vaultID, Name: name, Provider: provider,
		Notes: notes, RotateAt: rotateAt, CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (db *DB) ListKeys(vaultID string) ([]SecretKey, error) {
	rows, err := db.conn.Query(
		`SELECT id,vault_id,name,provider,notes,rotate_at,created_at,updated_at
		 FROM secret_keys WHERE vault_id=? ORDER BY name ASC`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SecretKey
	for rows.Next() {
		var k SecretKey
		rows.Scan(&k.ID, &k.VaultID, &k.Name, &k.Provider, &k.Notes, &k.RotateAt, &k.CreatedAt, &k.UpdatedAt)
		out = append(out, k)
	}
	return out, rows.Err()
}

func (db *DB) GetKeyValue(keyID string) (string, error) {
	var ciphertext string
	err := db.conn.QueryRow("SELECT ciphertext FROM secret_keys WHERE id=?", keyID).Scan(&ciphertext)
	if err != nil {
		return "", err
	}
	return db.decrypt(ciphertext)
}

func (db *DB) GetKeyByName(vaultID, name string) (*SecretKey, error) {
	var k SecretKey
	err := db.conn.QueryRow(
		`SELECT id,vault_id,name,provider,notes,rotate_at,created_at,updated_at
		 FROM secret_keys WHERE vault_id=? AND name=?`, vaultID, name).
		Scan(&k.ID, &k.VaultID, &k.Name, &k.Provider, &k.Notes, &k.RotateAt, &k.CreatedAt, &k.UpdatedAt)
	return &k, err
}

func (db *DB) RotateKey(keyID, newValue string) error {
	ciphertext, err := db.encrypt(newValue)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(
		"UPDATE secret_keys SET ciphertext=?, updated_at=datetime('now') WHERE id=?",
		ciphertext, keyID)
	return err
}

func (db *DB) DeleteKey(keyID string) error {
	_, err := db.conn.Exec("DELETE FROM secret_keys WHERE id=?", keyID)
	return err
}

// ── Members ───────────────────────────────────────────────────────────

type Member struct {
	ID        string `json:"id"`
	VaultID   string `json:"vault_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) AddMember(vaultID, username, role string) (*Member, error) {
	id := "mbr_" + genID(8)
	_, err := db.conn.Exec(
		"INSERT INTO members (id,vault_id,username,role) VALUES (?,?,?,?)",
		id, vaultID, username, role)
	if err != nil {
		return nil, err
	}
	return &Member{ID: id, VaultID: vaultID, Username: username, Role: role, Enabled: true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (db *DB) ListMembers(vaultID string) ([]Member, error) {
	rows, err := db.conn.Query(
		"SELECT id,vault_id,username,role,enabled,created_at FROM members WHERE vault_id=? ORDER BY created_at",
		vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		var en int
		rows.Scan(&m.ID, &m.VaultID, &m.Username, &m.Role, &en, &m.CreatedAt)
		m.Enabled = en == 1
		out = append(out, m)
	}
	return out, rows.Err()
}

func (db *DB) RevokeMember(memberID string) error {
	db.conn.Exec("UPDATE tokens SET enabled=0 WHERE member_id=?", memberID)
	_, err := db.conn.Exec("UPDATE members SET enabled=0 WHERE id=?", memberID)
	return err
}

func (db *DB) DeleteMember(memberID string) error {
	db.conn.Exec("DELETE FROM tokens WHERE member_id=?", memberID)
	_, err := db.conn.Exec("DELETE FROM members WHERE id=?", memberID)
	return err
}

// ── Tokens ────────────────────────────────────────────────────────────

type Token struct {
	ID        string `json:"id"`
	MemberID  string `json:"member_id"`
	VaultID   string `json:"vault_id"`
	KeyID     string `json:"key_id,omitempty"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

func (db *DB) IssueToken(memberID, vaultID, keyID, name string, ttlHours int) (rawToken string, t *Token, err error) {
	raw := make([]byte, 32)
	rand.Read(raw)
	rawToken = "fence_" + hex.EncodeToString(raw)

	h := sha256.Sum256([]byte(rawToken))
	hash := hex.EncodeToString(h[:])

	id := "tok_" + genID(8)
	expiresAt := ""
	if ttlHours > 0 {
		expiresAt = time.Now().Add(time.Duration(ttlHours) * time.Hour).UTC().Format("2006-01-02 15:04:05")
	}

	_, err = db.conn.Exec(
		`INSERT INTO tokens (id,member_id,vault_id,key_id,name,token_hash,expires_at) VALUES (?,?,?,?,?,?,?)`,
		id, memberID, vaultID, keyID, name, hash, expiresAt)
	if err != nil {
		return "", nil, err
	}
	return rawToken, &Token{ID: id, MemberID: memberID, VaultID: vaultID, KeyID: keyID,
		Name: name, ExpiresAt: expiresAt, Enabled: true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
}

func (db *DB) ListTokens(vaultID string) ([]Token, error) {
	rows, err := db.conn.Query(
		`SELECT id,member_id,vault_id,key_id,name,expires_at,enabled,created_at
		 FROM tokens WHERE vault_id=? ORDER BY created_at DESC`, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Token
	for rows.Next() {
		var t Token
		var en int
		rows.Scan(&t.ID, &t.MemberID, &t.VaultID, &t.KeyID, &t.Name, &t.ExpiresAt, &en, &t.CreatedAt)
		t.Enabled = en == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

// ResolveToken validates a raw token and returns the token record.
// Returns error if not found, disabled, or expired.
func (db *DB) ResolveToken(rawToken string) (*Token, error) {
	h := sha256.Sum256([]byte(rawToken))
	hash := hex.EncodeToString(h[:])

	var t Token
	var en int
	err := db.conn.QueryRow(
		`SELECT id,member_id,vault_id,key_id,name,expires_at,enabled,created_at
		 FROM tokens WHERE token_hash=?`, hash).
		Scan(&t.ID, &t.MemberID, &t.VaultID, &t.KeyID, &t.Name, &t.ExpiresAt, &en, &t.CreatedAt)
	if err != nil {
		return nil, errors.New("invalid token")
	}
	if en != 1 {
		return nil, errors.New("token revoked")
	}
	if t.ExpiresAt != "" {
		exp, err := time.Parse("2006-01-02 15:04:05", t.ExpiresAt)
		if err == nil && time.Now().After(exp) {
			return nil, errors.New("token expired")
		}
	}
	return &t, nil
}

func (db *DB) RevokeToken(tokenID string) error {
	_, err := db.conn.Exec("UPDATE tokens SET enabled=0 WHERE id=?", tokenID)
	return err
}

func (db *DB) DeleteToken(tokenID string) error {
	_, err := db.conn.Exec("DELETE FROM tokens WHERE id=?", tokenID)
	return err
}

// ── Access Log ────────────────────────────────────────────────────────

type AccessEntry struct {
	ID         int    `json:"id"`
	KeyID      string `json:"key_id"`
	KeyName    string `json:"key_name"`
	VaultID    string `json:"vault_id"`
	MemberID   string `json:"member_id"`
	TokenID    string `json:"token_id"`
	SourceIP   string `json:"source_ip"`
	AccessedAt string `json:"accessed_at"`
}

func (db *DB) LogAccess(keyID, keyName, vaultID, memberID, tokenID, ip string) {
	db.conn.Exec(
		`INSERT INTO access_log (key_id,key_name,vault_id,member_id,token_id,source_ip) VALUES (?,?,?,?,?,?)`,
		keyID, keyName, vaultID, memberID, tokenID, ip)
}

func (db *DB) ListAccess(vaultID string, limit int) ([]AccessEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := db.conn.Query(
		`SELECT id,key_id,key_name,vault_id,member_id,token_id,source_ip,accessed_at
		 FROM access_log WHERE vault_id=? ORDER BY accessed_at DESC LIMIT ?`, vaultID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessEntry
	for rows.Next() {
		var e AccessEntry
		rows.Scan(&e.ID, &e.KeyID, &e.KeyName, &e.VaultID, &e.MemberID, &e.TokenID, &e.SourceIP, &e.AccessedAt)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (db *DB) Stats() map[string]any {
	var vaults, keys, members, tokens, accesses int
	db.conn.QueryRow("SELECT COUNT(*) FROM vaults").Scan(&vaults)
	db.conn.QueryRow("SELECT COUNT(*) FROM secret_keys").Scan(&keys)
	db.conn.QueryRow("SELECT COUNT(*) FROM members WHERE enabled=1").Scan(&members)
	db.conn.QueryRow("SELECT COUNT(*) FROM tokens WHERE enabled=1").Scan(&tokens)
	db.conn.QueryRow("SELECT COUNT(*) FROM access_log").Scan(&accesses)
	return map[string]any{
		"vaults": vaults, "keys": keys,
		"members": members, "tokens": tokens, "accesses": accesses,
	}
}

// ── Helpers ───────────────────────────────────────────────────────────

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ExpiringKey is a key that is expiring soon.
type ExpiringKey struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	VaultID  string `json:"vault_id"`
	RotateAt string `json:"rotate_at"`
}

// KeysExpiringWithin returns keys whose rotate_at is within the next n days.
func (db *DB) KeysExpiringWithin(days int) []ExpiringKey {
	rows, err := db.conn.Query(`
		SELECT id, name, vault_id, rotate_at FROM secret_keys
		WHERE rotate_at IS NOT NULL AND rotate_at != ''
		AND rotate_at <= datetime('now', '+' || ? || ' days')
		AND rotate_at > datetime('now')
		ORDER BY rotate_at ASC`, days)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ExpiringKey
	for rows.Next() {
		var k ExpiringKey
		rows.Scan(&k.ID, &k.Name, &k.VaultID, &k.RotateAt)
		out = append(out, k)
	}
	return out
}
