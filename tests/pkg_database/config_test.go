package pkg_database_test

import (
	"os"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/pkg/database"
)

func TestConfig_Finalize_Defaults(t *testing.T) {
	cfg := &database.Config{
		Name: "testdb",
		User: "testuser",
	}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}

	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5432)
	}

	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 25)
	}

	if cfg.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 5)
	}

	if cfg.ConnMaxLifetime != "15m" {
		t.Errorf("ConnMaxLifetime = %q, want %q", cfg.ConnMaxLifetime, "15m")
	}

	if cfg.ConnTimeout != "5s" {
		t.Errorf("ConnTimeout = %q, want %q", cfg.ConnTimeout, "5s")
	}
}

func TestConfig_Finalize_PreservesValues(t *testing.T) {
	cfg := &database.Config{
		Host:            "dbhost",
		Port:            5433,
		Name:            "mydb",
		User:            "myuser",
		Password:        "secret",
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: "30m",
		ConnTimeout:     "10s",
	}

	if err := cfg.Finalize(nil); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if cfg.Host != "dbhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "dbhost")
	}

	if cfg.Port != 5433 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5433)
	}

	if cfg.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 50)
	}

	if cfg.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 10)
	}

	if cfg.ConnMaxLifetime != "30m" {
		t.Errorf("ConnMaxLifetime = %q, want %q", cfg.ConnMaxLifetime, "30m")
	}

	if cfg.ConnTimeout != "10s" {
		t.Errorf("ConnTimeout = %q, want %q", cfg.ConnTimeout, "10s")
	}
}

func TestConfig_Finalize_EnvOverrides(t *testing.T) {
	os.Setenv("TEST_DB_HOST", "envhost")
	os.Setenv("TEST_DB_PORT", "5434")
	os.Setenv("TEST_DB_NAME", "envdb")
	os.Setenv("TEST_DB_USER", "envuser")
	os.Setenv("TEST_DB_PASSWORD", "envsecret")
	os.Setenv("TEST_DB_MAX_OPEN", "100")
	os.Setenv("TEST_DB_MAX_IDLE", "20")
	os.Setenv("TEST_DB_LIFETIME", "1h")
	os.Setenv("TEST_DB_TIMEOUT", "30s")
	defer func() {
		os.Unsetenv("TEST_DB_HOST")
		os.Unsetenv("TEST_DB_PORT")
		os.Unsetenv("TEST_DB_NAME")
		os.Unsetenv("TEST_DB_USER")
		os.Unsetenv("TEST_DB_PASSWORD")
		os.Unsetenv("TEST_DB_MAX_OPEN")
		os.Unsetenv("TEST_DB_MAX_IDLE")
		os.Unsetenv("TEST_DB_LIFETIME")
		os.Unsetenv("TEST_DB_TIMEOUT")
	}()

	cfg := &database.Config{}
	env := &database.Env{
		Host:            "TEST_DB_HOST",
		Port:            "TEST_DB_PORT",
		Name:            "TEST_DB_NAME",
		User:            "TEST_DB_USER",
		Password:        "TEST_DB_PASSWORD",
		MaxOpenConns:    "TEST_DB_MAX_OPEN",
		MaxIdleConns:    "TEST_DB_MAX_IDLE",
		ConnMaxLifetime: "TEST_DB_LIFETIME",
		ConnTimeout:     "TEST_DB_TIMEOUT",
	}

	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	if cfg.Host != "envhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "envhost")
	}

	if cfg.Port != 5434 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5434)
	}

	if cfg.Name != "envdb" {
		t.Errorf("Name = %q, want %q", cfg.Name, "envdb")
	}

	if cfg.User != "envuser" {
		t.Errorf("User = %q, want %q", cfg.User, "envuser")
	}

	if cfg.Password != "envsecret" {
		t.Errorf("Password = %q, want %q", cfg.Password, "envsecret")
	}

	if cfg.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", cfg.MaxOpenConns, 100)
	}

	if cfg.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns = %d, want %d", cfg.MaxIdleConns, 20)
	}

	if cfg.ConnMaxLifetime != "1h" {
		t.Errorf("ConnMaxLifetime = %q, want %q", cfg.ConnMaxLifetime, "1h")
	}

	if cfg.ConnTimeout != "30s" {
		t.Errorf("ConnTimeout = %q, want %q", cfg.ConnTimeout, "30s")
	}
}

func TestConfig_Finalize_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     database.Config
		wantErr string
	}{
		{
			name:    "missing name",
			cfg:     database.Config{User: "user"},
			wantErr: "name required",
		},
		{
			name:    "missing user",
			cfg:     database.Config{Name: "db"},
			wantErr: "user required",
		},
		{
			name:    "invalid conn_max_lifetime",
			cfg:     database.Config{Name: "db", User: "user", ConnMaxLifetime: "invalid"},
			wantErr: "invalid conn_max_lifetime",
		},
		{
			name:    "invalid conn_timeout",
			cfg:     database.Config{Name: "db", User: "user", ConnTimeout: "invalid"},
			wantErr: "invalid conn_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Finalize(nil)
			if err == nil {
				t.Fatal("Finalize() should return error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	tests := []struct {
		name    string
		base    database.Config
		overlay database.Config
		want    database.Config
	}{
		{
			name: "overlay overrides all",
			base: database.Config{
				Host:         "localhost",
				Port:         5432,
				Name:         "db1",
				User:         "user1",
				Password:     "pass1",
				MaxOpenConns: 25,
				MaxIdleConns: 5,
			},
			overlay: database.Config{
				Host:         "remotehost",
				Port:         5433,
				Name:         "db2",
				User:         "user2",
				Password:     "pass2",
				MaxOpenConns: 50,
				MaxIdleConns: 10,
			},
			want: database.Config{
				Host:         "remotehost",
				Port:         5433,
				Name:         "db2",
				User:         "user2",
				Password:     "pass2",
				MaxOpenConns: 50,
				MaxIdleConns: 10,
			},
		},
		{
			name: "zero values preserve base",
			base: database.Config{
				Host:         "localhost",
				Port:         5432,
				Name:         "db1",
				User:         "user1",
				MaxOpenConns: 25,
			},
			overlay: database.Config{},
			want: database.Config{
				Host:         "localhost",
				Port:         5432,
				Name:         "db1",
				User:         "user1",
				MaxOpenConns: 25,
			},
		},
		{
			name: "partial overlay",
			base: database.Config{
				Host:         "localhost",
				Port:         5432,
				Name:         "db1",
				User:         "user1",
				MaxOpenConns: 25,
			},
			overlay: database.Config{
				Host: "remotehost",
				Port: 5433,
			},
			want: database.Config{
				Host:         "remotehost",
				Port:         5433,
				Name:         "db1",
				User:         "user1",
				MaxOpenConns: 25,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(&tt.overlay)

			if tt.base.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", tt.base.Host, tt.want.Host)
			}

			if tt.base.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", tt.base.Port, tt.want.Port)
			}

			if tt.base.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", tt.base.Name, tt.want.Name)
			}

			if tt.base.User != tt.want.User {
				t.Errorf("User = %q, want %q", tt.base.User, tt.want.User)
			}

			if tt.base.Password != tt.want.Password {
				t.Errorf("Password = %q, want %q", tt.base.Password, tt.want.Password)
			}

			if tt.base.MaxOpenConns != tt.want.MaxOpenConns {
				t.Errorf("MaxOpenConns = %d, want %d", tt.base.MaxOpenConns, tt.want.MaxOpenConns)
			}

			if tt.base.MaxIdleConns != tt.want.MaxIdleConns {
				t.Errorf("MaxIdleConns = %d, want %d", tt.base.MaxIdleConns, tt.want.MaxIdleConns)
			}
		})
	}
}

func TestConfig_Dsn(t *testing.T) {
	cfg := &database.Config{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "secret",
	}

	dsn := cfg.Dsn()

	expected := "host=localhost port=5432 dbname=testdb user=testuser password=secret sslmode=disable"
	if dsn != expected {
		t.Errorf("Dsn() = %q, want %q", dsn, expected)
	}
}

func TestConfig_ConnMaxLifetimeDuration(t *testing.T) {
	cfg := &database.Config{
		ConnMaxLifetime: "30m",
	}

	d := cfg.ConnMaxLifetimeDuration()

	if d.Minutes() != 30 {
		t.Errorf("ConnMaxLifetimeDuration() = %v, want 30m", d)
	}
}

func TestConfig_ConnTimeoutDuration(t *testing.T) {
	cfg := &database.Config{
		ConnTimeout: "10s",
	}

	d := cfg.ConnTimeoutDuration()

	if d.Seconds() != 10 {
		t.Errorf("ConnTimeoutDuration() = %v, want 10s", d)
	}
}
