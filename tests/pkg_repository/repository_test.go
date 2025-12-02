package pkg_repository_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	errNotFound  = errors.New("not found")
	errDuplicate = errors.New("duplicate")
)

func TestMapError_Nil(t *testing.T) {
	result := repository.MapError(nil, errNotFound, errDuplicate)
	if result != nil {
		t.Errorf("MapError(nil) = %v, want nil", result)
	}
}

func TestMapError_NoRows(t *testing.T) {
	result := repository.MapError(sql.ErrNoRows, errNotFound, errDuplicate)
	if !errors.Is(result, errNotFound) {
		t.Errorf("MapError(sql.ErrNoRows) = %v, want %v", result, errNotFound)
	}
}

func TestMapError_PgDuplicate(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	result := repository.MapError(pgErr, errNotFound, errDuplicate)
	if !errors.Is(result, errDuplicate) {
		t.Errorf("MapError(pgErr 23505) = %v, want %v", result, errDuplicate)
	}
}

func TestMapError_OtherPgError(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "12345"}
	result := repository.MapError(pgErr, errNotFound, errDuplicate)
	if result != pgErr {
		t.Errorf("MapError(pgErr other) = %v, want original error", result)
	}
}

func TestMapError_OtherError(t *testing.T) {
	otherErr := errors.New("some other error")
	result := repository.MapError(otherErr, errNotFound, errDuplicate)
	if result != otherErr {
		t.Errorf("MapError(otherErr) = %v, want original error", result)
	}
}
