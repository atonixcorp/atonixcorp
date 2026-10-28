package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type projectAuthorizer interface {
	AuthorizeProject(userID, tenantID, projectID string) (bool, error)
}

type claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

type authenticatedUserKey struct{}

func authenticate(jwtSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(jwtSecret) == "" {
			writeError(w, http.StatusServiceUnavailable, "AUTHENTICATION_UNAVAILABLE", "Authentication is not configured.")
			return
		}
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication is required.")
			return
		}
		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) {
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Authentication is required.")
			return
		}
		userClaims, ok := token.Claims.(*claims)
		if !ok || !token.Valid || userClaims.UserID == 0 {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Authentication is required.")
			return
		}
		ctx := context.WithValue(r.Context(), authenticatedUserKey{}, userClaims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func authorizeProject(authorizer projectAuthorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(authenticatedUserKey{}).(uint)
		vars := mux.Vars(r)
		tenantID, projectID := vars["tenantID"], vars["projectID"]
		if !ok || !validPositiveID(tenantID) || !validPositiveID(projectID) {
			writeError(w, http.StatusBadRequest, "INVALID_SCOPE", "A valid tenant and project are required.")
			return
		}
		allowed, err := authorizer.AuthorizeProject(
			strconv.FormatUint(uint64(userID), 10), tenantID, projectID,
		)
		if err != nil {
			log.Printf("authorize networking project access: %v", err)
			writeError(w, http.StatusInternalServerError, "AUTHORIZATION_FAILED", "Project access could not be verified.")
			return
		}
		if !allowed {
			writeError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "Project not found.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validPositiveID(value string) bool {
	id, err := strconv.ParseUint(value, 10, 64)
	return err == nil && id > 0
}
