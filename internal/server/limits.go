package server

import "github.com/stockyard-dev/stockyard-fence/internal/license"

// Limits holds the feature limits for the current license tier.
// All int limits: 0 means unlimited (Pro tier only).
type Limits struct {
	MaxKeys int // 0 = unlimited (Pro)
	MaxMembers int // 0 = unlimited (Pro)
	MaxVaults int // 0 = unlimited (Pro)
	RBACRoles bool
	FullAuditTrail bool
	ExpirationReminders bool
	ExportImport bool
}

var freeLimits = Limits{
		MaxKeys: 10,
		MaxMembers: 2,
		MaxVaults: 2,
		RBACRoles: false,
		FullAuditTrail: false,
		ExpirationReminders: false,
		ExportImport: false,
}

var proLimits = Limits{
		MaxKeys: 0,
		MaxMembers: 0,
		MaxVaults: 0,
		RBACRoles: true,
		FullAuditTrail: true,
		ExpirationReminders: true,
		ExportImport: true,
}

// LimitsFor returns the appropriate Limits for the given license info.
// nil info = no key set = free tier.
func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

// LimitReached returns true if the current count meets or exceeds the limit.
// A limit of 0 is treated as unlimited.
func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
