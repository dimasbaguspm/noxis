package entity

// The HTTP contract.

// AuthConfig is the public auth configuration the console bootstraps from: which methods are available and, when OIDC is on, the issuer and client id (neither is a secret).
type AuthConfig struct {
	Local bool        `json:"local"`
	OIDC  *OIDCConfig `json:"oidc,omitempty"`
}

// OIDCConfig is the public OIDC client configuration for the console's PKCE flow.
type OIDCConfig struct {
	Issuer   string `json:"issuer"`
	ClientID string `json:"clientId"`
}

// LoginRequest is the local credentials payload.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the token the console stores and the account it belongs to.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// PasswordChange is the self-serve password payload.
type PasswordChange struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// AccessGrant is the caller's effective level on a connection and when it lapses.
type AccessGrant struct {
	Level     string `json:"level"`
	ExpiresAt int64  `json:"expiresAt"`
}

// MeResponse is the caller plus the effective grant it holds on each readable connection, so the console can gate actions without guessing.
type MeResponse struct {
	Identity
	Connections map[string]AccessGrant `json:"connections"`
}

// ConnectionList is the connection collection response.
type ConnectionList struct {
	Connections []Connection `json:"connections"`
}

// InstanceList is a page of instances and the size of the whole set.
type InstanceList struct {
	Instances []Connection `json:"instances"`
	Total     int          `json:"total"`
}

// InstancePassword is the rotate-password payload.
type InstancePassword struct {
	Password string `json:"password"`
}

// ChangeList is a page of changes and the size of the whole set.
type ChangeList struct {
	Changes []Change `json:"changes"`
	Total   int      `json:"total"`
}

// MongoChangeRequest is one submitted MongoDB request: the connection and notes of a change, plus the request itself, flat so the console sends one object.
type MongoChangeRequest struct {
	Connection string `json:"connection"`
	Notes      string `json:"notes"`
	MongoRequest
}

// ScheduleRequest is the schedule payload: unix seconds, 0 clears the schedule.
type ScheduleRequest struct {
	ScheduledAt int64 `json:"scheduledAt"`
}

// TickResult is one scheduler pass: the ids it applied and the ids it failed to apply (the failure is already recorded on the change itself).
type TickResult struct {
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
}

// PermissionList is the permission collection response.
type PermissionList struct {
	Permissions []Permission `json:"permissions"`
}

// PermissionGrant is the upsert payload: one member at one level.
type PermissionGrant struct {
	Member     string `json:"member"`
	MemberType string `json:"memberType"`
	Level      string `json:"level"`
}

// Member is one pickable account: the subject a permission stores and a human label for the console.
type Member struct {
	Subject string `json:"subject"`
	Label   string `json:"label"`
	Role    string `json:"role"`
}

// MemberList is the known groups and accounts for permission editing.
type MemberList struct {
	Groups []string `json:"groups"`
	Users  []Member `json:"users"`
}

// InstanceAccess is one instance with its whole access policy: who holds what, and until when.
type InstanceAccess struct {
	ID          string       `json:"id"`
	Connection  string       `json:"connection"`
	Driver      string       `json:"driver"`
	Permissions []Permission `json:"permissions"`
}

// InstanceAccessList is the access-overview response.
type InstanceAccessList struct {
	Instances []InstanceAccess `json:"instances"`
}

// AccessRequestList is the access-request collection response.
type AccessRequestList struct {
	Requests []AccessRequest `json:"requests"`
}

// AccessRequestCreate files an access request.
type AccessRequestCreate struct {
	Connection string `json:"connection"`
	Level      string `json:"level"`
	Reason     string `json:"reason"`
}

// AccessDecision is the approve/reject payload.
type AccessDecision struct {
	Note string `json:"note"`
}

// CacheDeleteRequest is the remove-keys payload.
type CacheDeleteRequest struct {
	Keys []string `json:"keys"`
}

// CacheDeleteResult reports how many keys were removed.
type CacheDeleteResult struct {
	Deleted int64 `json:"deleted"`
}

// CacheFlushRequest is the flush payload: Confirm must repeat the connection name.
type CacheFlushRequest struct {
	Confirm string `json:"confirm"`
}

// DatabaseList is the MongoDB database collection response.
type DatabaseList struct {
	Databases []Database `json:"databases"`
}

// CollectionList is the MongoDB collection collection response.
type CollectionList struct {
	Collections []Collection `json:"collections"`
}

// TableList is the SQL schema response.
type TableList struct {
	Tables []Table `json:"tables"`
}

// OpRequest is the SQL console op payload: op plus sql for query/alter/rollback, table for analyze.
type OpRequest struct {
	Op    string `json:"op"`
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// LevelInfo is one access level and its human label.
type LevelInfo struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// LevelList is the access-level catalog response.
type LevelList struct {
	Levels []LevelInfo `json:"levels"`
}

// DriverInfo is one entry in the server-owned driver picker catalog.
type DriverInfo struct {
	Driver string `json:"driver"`
	Label  string `json:"label"`
}

// UserList is the account directory response.
type UserList struct {
	Users []User `json:"users"`
}

// UserDetail is one account with the access it holds and its activity trail.
type UserDetail struct {
	User
	Access []Permission  `json:"access"`
	Events []AccessEvent `json:"events"`
}

// UserUpdate is the account patch payload. Both fields are optional; the caller changes only what it sends.
type UserUpdate struct {
	Status *string `json:"status"`
	Role   *string `json:"role"`
}
