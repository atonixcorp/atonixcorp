package gatewayapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	platformconfig "atonixcorp/devops/internal/platform/config"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var errLocalAuthDisabled = errors.New("local authentication is disabled")

type localAuthAccount struct {
	ID           string
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
	Role         string
	Status       string
	Identity     string
}

func localAuthEnabled() bool {
	return platformconfig.Bool("ORCASTACK_AUTH_LOCAL_ENABLED", false)
}

func localAuthAutoApprove() bool {
	return platformconfig.Bool("ORCASTACK_AUTH_LOCAL_AUTO_APPROVE", true)
}

func authenticateUser(username, password string) (AuthUser, error) {
	if localAuthEnabled() {
		user, err := authenticateLocalUser(username, password)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, errInvalidCredentials) {
			return AuthUser{}, err
		}
	}

	user, err := authenticateLDAPUser(username, password)
	if err == nil {
		return user, nil
	}
	if localAuthEnabled() && strings.Contains(strings.ToLower(err.Error()), "ldap authentication is not configured") {
		return AuthUser{}, errInvalidCredentials
	}

	return AuthUser{}, err
}

func authenticateLocalUser(username, password string) (AuthUser, error) {
	if !localAuthEnabled() {
		return AuthUser{}, errLocalAuthDisabled
	}

	account, err := loadLocalAuthAccount(context.Background(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthUser{}, errInvalidCredentials
		}
		return AuthUser{}, err
	}

	if account.Status != "approved" {
		return AuthUser{}, errors.New("account access is pending administrator approval")
	}

	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return AuthUser{}, errInvalidCredentials
	}

	permissions := []string{"repositories:read", "pipelines:write", "deployments:write", "control-panel:read", "community:read"}
	if account.Role == "platform-admin" {
		permissions = []string{"repositories:read", "pipelines:write", "deployments:write", "control-panel:admin", "community:read"}
	}

	return AuthUser{
		Username:    account.Username,
		FullName:    account.DisplayName,
		Email:       account.Email,
		Role:        account.Role,
		Identity:    account.Identity,
		RBACRealm:   platformconfig.String("ORCASTACK_AUTH_DEFAULT_REALM", platformconfig.String("ORCASTACK_RBAC_REALM", "platform")),
		Permissions: permissions,
	}, nil
}

func createLocalAuthAccount(ctx context.Context, requestID, username, email, password string) (string, error) {
	if !localAuthEnabled() {
		return "pending_review", nil
	}

	db, err := signupDB()
	if err != nil {
		return "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash local account password: %w", err)
	}

	status := "pending_review"
	if localAuthAutoApprove() {
		status = "approved"
	}

	_, err = db.ExecContext(ctx, `
		insert into local_auth_accounts (id, username, email, display_name, password_hash, role, status, identity)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, requestID, username, email, username, string(passwordHash), "platform-operator", status, "local:user:"+username)
	if err != nil {
		return "", fmt.Errorf("insert local auth account: %w", err)
	}

	return status, nil
}

func loadLocalAuthAccount(ctx context.Context, username string) (localAuthAccount, error) {
	db, err := signupDB()
	if err != nil {
		return localAuthAccount{}, err
	}

	var account localAuthAccount
	err = db.QueryRowContext(ctx, `
		select id, username, email, display_name, password_hash, role, status, identity
		from local_auth_accounts
		where lower(username) = lower($1) or lower(email) = lower($1)
	`, username).Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.DisplayName,
		&account.PasswordHash,
		&account.Role,
		&account.Status,
		&account.Identity,
	)
	if err != nil {
		return localAuthAccount{}, err
	}

	return account, nil
}

func syncLocalAuthAccountReview(ctx context.Context, requestID, decision string) error {
	if !localAuthEnabled() {
		return nil
	}

	db, err := signupDB()
	if err != nil {
		return err
	}

	status := decision
	if status == "pending_review" {
		status = "pending_review"
	}

	_, err = db.ExecContext(ctx, `update local_auth_accounts set status = $2 where id = $1`, requestID, status)
	if err != nil {
		return fmt.Errorf("sync local auth account review: %w", err)
	}

	return nil
}

func ensureConfiguredLocalAdminAccount(ctx context.Context) error {
	if !localAuthEnabled() {
		return nil
	}

	username := strings.TrimSpace(platformconfig.String("ORCASTACK_AUTH_LOCAL_ADMIN_USERNAME", ""))
	password := strings.TrimSpace(platformconfig.String("ORCASTACK_AUTH_LOCAL_ADMIN_PASSWORD", ""))
	email := strings.TrimSpace(platformconfig.String("ORCASTACK_AUTH_LOCAL_ADMIN_EMAIL", ""))
	if username == "" || password == "" || email == "" {
		return errors.New("local admin username, email, and password must be configured when local auth is enabled")
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("local admin password must be between 12 and 72 bytes")
	}

	if _, err := loadLocalAuthAccount(ctx, username); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	db, err := signupDB()
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash configured local admin password: %w", err)
	}

	adminID := uuid.NewString()
	_, err = db.ExecContext(ctx, `
		insert into local_auth_accounts (id, username, email, display_name, password_hash, role, status, identity)
		values ($1, $2, $3, $4, $5, 'platform-admin', 'approved', $6)
		on conflict (username) do nothing
	`, adminID, username, email, "Local Administrator", string(passwordHash), "local:user:"+username)
	if err != nil {
		return fmt.Errorf("insert configured local admin account: %w", err)
	}

	return nil
}
