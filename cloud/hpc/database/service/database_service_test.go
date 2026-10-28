package service

import (
	"testing"

	"atonixcorp/cloud/hpc/database/model"
)

func TestManagedDatabaseRegistrationRequiresSecretReferenceAndTLS(t *testing.T) {
	db := NewDatabaseService()
	database := model.ManagedDatabase{
		ID: "db-1", Engine: "postgres", Endpoint: "10.0.2.15",
		Port: 5432, DatabaseName: "app", CredentialRef: "secret://db-1/admin",
		TLSRequired: true, NetworkID: "private-db-network",
	}
	if err := db.RegisterManagedDatabase(database); err != nil {
		t.Fatalf("register private managed database: %v", err)
	}
	registered, ok := db.ManagedDatabase(database.ID)
	if !ok || registered != database {
		t.Fatalf("registered database = %+v, %t", registered, ok)
	}

	database.ID = "db-no-secret"
	database.CredentialRef = ""
	if err := db.RegisterManagedDatabase(database); err == nil {
		t.Fatal("expected missing secret reference to fail")
	}
	database.CredentialRef = "secret://db-1/admin"
	database.TLSRequired = false
	if err := db.RegisterManagedDatabase(database); err == nil {
		t.Fatal("expected non-TLS database registration to fail")
	}
}

func TestRuntimeIntegrationLooksUpManagedDatabaseForSlurm(t *testing.T) {
	db := NewDatabaseService()
	runtime := NewRuntimeIntegration(db)
	if _, err := runtime.DatabaseForSlurm("missing"); err == nil {
		t.Fatal("expected missing database binding to fail")
	}
	expected := model.ManagedDatabase{
		ID: "db-1", Engine: "postgres", Endpoint: "10.0.2.15",
		Port: 5432, DatabaseName: "simulation", CredentialRef: "secret://db-1/user",
		TLSRequired: true, NetworkID: "hpc-private-network",
	}
	if err := db.RegisterManagedDatabase(expected); err != nil {
		t.Fatal(err)
	}
	got, err := runtime.DatabaseForSlurm(expected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("database binding = %+v, want %+v", got, expected)
	}
}
