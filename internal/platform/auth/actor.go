package auth

// Actor is the authenticated application identity used by handlers and services.
// UserID is authoritative from the active database session, while roles and
// permissions are the effective capabilities resolved for that session.
type Actor struct {
	UserID      uint
	Roles       []string
	Permissions []string
}

func (a Actor) HasRole(role string) bool {
	for _, value := range a.Roles {
		if value == role {
			return true
		}
	}
	return false
}

func (a Actor) HasPermission(permission string) bool {
	for _, value := range a.Permissions {
		if value == permission {
			return true
		}
	}
	return false
}
