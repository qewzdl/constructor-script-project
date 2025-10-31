package authorization

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
	RoleAuthor UserRole = "author"
	RoleUser   UserRole = "user"
)

var validRoles = map[UserRole]struct{}{
	RoleAdmin:  {},
	RoleEditor: {},
	RoleAuthor: {},
	RoleUser:   {},
}

func (r UserRole) String() string {
	return string(r)
}

func (r UserRole) IsValid() bool {
	_, ok := validRoles[r]
	return ok
}

func (r UserRole) Value() (driver.Value, error) {
	if r == "" {
		return string(RoleUser), nil
	}
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid user role: %q", r)
	}
	return string(r), nil
}

func (r *UserRole) Scan(value interface{}) error {
	if value == nil {
		*r = RoleUser
		return nil
	}

	switch v := value.(type) {
	case string:
		role := UserRole(strings.ToLower(strings.TrimSpace(v)))
		if !role.IsValid() {
			return fmt.Errorf("invalid user role: %q", v)
		}
		*r = role
		return nil
	case []byte:
		role := UserRole(strings.ToLower(strings.TrimSpace(string(v))))
		if !role.IsValid() {
			return fmt.Errorf("invalid user role: %q", v)
		}
		*r = role
		return nil
	default:
		return fmt.Errorf("unsupported type for UserRole: %T", value)
	}
}

type Permission string

const (
	PermissionManageUsers        Permission = "manage_users"
	PermissionManageAllContent   Permission = "manage_all_content"
	PermissionManageOwnContent   Permission = "manage_own_content"
	PermissionPublishContent     Permission = "publish_content"
	PermissionModerateComments   Permission = "moderate_comments"
	PermissionManageSettings     Permission = "manage_settings"
	PermissionManageThemes       Permission = "manage_themes"
	PermissionManagePlugins      Permission = "manage_plugins"
	PermissionManageBackups      Permission = "manage_backups"
	PermissionManageNavigation   Permission = "manage_navigation"
	PermissionManageIntegrations Permission = "manage_integrations"
)

var rolePermissions = map[UserRole]map[Permission]struct{}{
	RoleAdmin: {
		PermissionManageUsers:        {},
		PermissionManageAllContent:   {},
		PermissionPublishContent:     {},
		PermissionModerateComments:   {},
		PermissionManageSettings:     {},
		PermissionManageThemes:       {},
		PermissionManagePlugins:      {},
		PermissionManageBackups:      {},
		PermissionManageNavigation:   {},
		PermissionManageIntegrations: {},
	},
	RoleEditor: {
		PermissionManageAllContent: {},
		PermissionPublishContent:   {},
		PermissionModerateComments: {},
	},
	RoleAuthor: {
		PermissionManageOwnContent: {},
	},
	RoleUser: {},
}

func RoleHasPermission(role UserRole, permission Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[permission]
	return ok
}

func ParseUserRole(value interface{}) (UserRole, bool) {
	switch v := value.(type) {
	case UserRole:
		if !v.IsValid() {
			return "", false
		}
		return v, true
	case string:
		role := UserRole(strings.ToLower(strings.TrimSpace(v)))
		if !role.IsValid() {
			return "", false
		}
		return role, true
	case []byte:
		role := UserRole(strings.ToLower(strings.TrimSpace(string(v))))
		if !role.IsValid() {
			return "", false
		}
		return role, true
	default:
		return "", false
	}
}

func ValidRoles() []UserRole {
	roles := make([]UserRole, 0, len(validRoles))
	for role := range validRoles {
		roles = append(roles, role)
	}
	return roles
}
