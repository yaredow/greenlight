package data

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/lib/pq"
	"github.com/yaredow/greenlight/internal/validator"
)

type PermissionModel struct {
	DB *sql.DB
}

type Permissions []string

var (
	PermissionMoviesRead  = "movies:read"
	PermissionMoviesWrite = "movies:write"
)

var SupportedPermissions = Permissions{
	PermissionMoviesRead,
	PermissionMoviesWrite,
}

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

func ValidatePermissions(v *validator.Validator, permissions Permissions) {
	v.Check(len(permissions) > 0, "permissions", "must contain at least one permission")

	for _, p := range permissions {
		v.Check(slices.Contains(SupportedPermissions, p), "permissions", fmt.Sprintf("invalid permission: %s", p))
	}
}

func (m *PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		WHERE users_permissions.user_id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (m *PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO users_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID, pq.Array(codes))
	return err
}
