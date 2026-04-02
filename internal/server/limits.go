package server

type Limits struct {
	MaxKeys int // 0 = unlimited (Pro)
	MaxMembers int // 0 = unlimited (Pro)
	MaxVaults int // 0 = unlimited (Pro)
	RBACRoles bool
	FullAuditTrail bool
	ExpirationReminders bool
	ExportImport bool
}

// DefaultLimits returns fully-unlocked limits for the standalone edition.
func DefaultLimits() Limits {
	return Limits{
		MaxKeys: 0,
		MaxMembers: 0,
		MaxVaults: 0,
		RBACRoles: true,
		FullAuditTrail: true,
		ExpirationReminders: true,
		ExportImport: true,
}
}

// LimitReached returns true if the current count meets or exceeds the limit.
// A limit of 0 is treated as unlimited.
func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
