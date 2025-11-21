# Milestone 1: Provider & Agent Configuration Management

## Problem Context

agent-lab requires a foundation for managing LLM provider and agent configurations with:
- Database-backed persistence of configurations
- RESTful API for CRUD operations
- Robust pagination, search, filtering, and ordering for multi-record queries
- Protocol execution endpoints for testing agent connectivity
- Interactive API documentation
- Validation using go-agents configuration structures

This milestone establishes the core infrastructure and patterns that will be extended in future milestones.

## Architecture Approach

### Service Lifecycle Model

**Long-Running Services** (Application-scoped):
- Database connection pool
- Logger
- PaginationService
- Stored in `Application` struct

**Ephemeral Services** (Request-scoped):
- ProviderService
- AgentService
- Initialized per HTTP request with request context
- Never create consolidated "Services" struct

### Data Boundary Transformation

All request processing follows: **Finalize → Validate → Transform**

Encapsulated in constructor methods (`FromRequest()`):
1. **Finalize**: Parse request into go-agents config structures
2. **Validate**: Attempt to create provider/agent to validate config
3. **Transform**: Create domain model for persistence

### Query Engine Architecture

Three-layer separation inspired by S2va pattern:

**Layer 1: ProjectionMap** - Declarative query structure
- Defines tables, joins, column mappings
- Static, reusable across service methods
- Resolves view property names to table.column references

**Layer 2: QueryBuilder** - Fluent query operations
- Filters (equals, contains, search)
- Sorting with defaults
- Pagination (count + page queries)
- Automatic parameter management

**Layer 3: Execution** - database/sql operations
- Use generated SQL + args with QueryContext/ExecContext
- Context propagation for cancellation and timeouts

### Protocol Execution Pattern

For each agent protocol method:
1. Load agent config from database
2. Extract Bearer token from Authorization header (for Entra auth)
3. Inject token into config if auth_type is "bearer"
4. Create ephemeral agent using go-agents
5. Execute protocol method
6. Return protocol-specific response

### Pagination Strategy

- Default page size: 50
- Maximum page size: 100
- Smart adjustment: clamp page to valid range
- Two-query pattern: COUNT for total, then SELECT with OFFSET/FETCH
- Validation service applies defaults and constraints

## Implementation Steps

### Phase 1: Foundation Infrastructure

#### Step 1.1: Initialize Go Module

```bash
go mod init github.com/JaimeStill/agent-lab
```

**go.mod**:
```go
module github.com/JaimeStill/agent-lab

go 1.25

require (
	github.com/JaimeStill/go-agents v0.2.1
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.2
)
```

#### Step 1.2: Project Structure

Create directory structure:
```bash
mkdir -p cmd/server
mkdir -p internal/config
mkdir -p internal/models
mkdir -p internal/query
mkdir -p internal/services
mkdir -p internal/handlers
mkdir -p migrations
mkdir -p api
mkdir -p web/scalar
```

Install TOML configuration library:
```bash
go get github.com/pelletier/go-toml/v2
```

#### Step 1.3: Docker Compose Setup

Create modular compose structure for flexibility:

```bash
mkdir -p compose
```

**compose/postgres.yml**:
```yaml
services:
  postgres:
    image: postgres:17
    container_name: agent-lab-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-agent_lab}
      POSTGRES_USER: ${POSTGRES_USER:-agent_lab}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-agent_lab}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-agent_lab}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - agent-lab

volumes:
  postgres-data:

networks:
  agent-lab:
    name: agent-lab
    driver: bridge
```

**compose/ollama.yml**:
```yaml
services:
  ollama:
    image: ollama/ollama:latest
    container_name: agent-lab-ollama
    ports:
      - "${OLLAMA_PORT:-11434}:11434"
    volumes:
      - ${OLLAMA_MODELS_DIR:-~/.ollama}:/root/.ollama
    environment:
      - OLLAMA_KEEP_ALIVE=${OLLAMA_KEEP_ALIVE:-5m}
      - OLLAMA_HOST=0.0.0.0
    devices:
      - nvidia.com/gpu=all
    restart: unless-stopped
    networks:
      - agent-lab

networks:
  agent-lab:
    name: agent-lab
    external: true
```

**docker-compose.yml** (base - postgres only):
```yaml
include:
  - compose/postgres.yml
```

**docker-compose.dev.yml** (development with Ollama):
```yaml
include:
  - compose/postgres.yml
  - compose/ollama.yml
```

**.env**:
```env
POSTGRES_DB=agent_lab
POSTGRES_USER=agent_lab
POSTGRES_PASSWORD=agent_lab
POSTGRES_PORT=5432

OLLAMA_PORT=11434
OLLAMA_MODELS_DIR=~/.ollama
OLLAMA_KEEP_ALIVE=5m

SERVER_PORT=8080
```

**Note**: Ollama service starts without any models. Pull models manually as needed:

```bash
# Pull llama3.2:3b (recommended for testing)
docker exec -it agent-lab-ollama ollama pull llama3.2:3b

# List available models
docker exec -it agent-lab-ollama ollama list

# Pull other models as needed
docker exec -it agent-lab-ollama ollama pull mistral
```

#### Step 1.4: Database Migration

**migrations/001_initial_schema.sql**:
```sql
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_providers_name ON providers(name);
CREATE INDEX idx_agents_name ON agents(name);
```

#### Step 1.5: Application Configuration

**config.toml**:
```toml
# HTTP server configuration
[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
idle_timeout = "120s"
shutdown_timeout = "15s"

# PostgreSQL connection settings
[database]
host = "localhost"
port = 5432
database = "agent_lab"
user = "agent_lab"
password = ""
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = "15m"

# Pagination defaults and limits
[pagination]
default_page_size = 50
max_page_size = 100

# CORS configuration for web clients
[cors]
origins = ["http://localhost:3000"]
methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
headers = ["Content-Type", "Authorization"]
credentials = true

# Logging configuration
[logging]
level = "info"
format = "text"
```

**.env.example**:
```env
SERVER_PORT=8080

DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_DATABASE=agent_lab
DATABASE_USER=agent_lab
DATABASE_PASSWORD=agent_lab

PAGINATION_DEFAULT_PAGE_SIZE=50
PAGINATION_MAX_PAGE_SIZE=100

LOGGING_LEVEL=info
```

#### Step 1.6: Configuration Loading

**internal/config/config.go**:
```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server     ServerConfig     `toml:"server"`
	Database   DatabaseConfig   `toml:"database"`
	Pagination PaginationConfig `toml:"pagination"`
	CORS       CORSConfig       `toml:"cors"`
	Logging    LoggingConfig    `toml:"logging"`
}

type ServerConfig struct {
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	ReadTimeout     time.Duration `toml:"read_timeout"`
	WriteTimeout    time.Duration `toml:"write_timeout"`
	IdleTimeout     time.Duration `toml:"idle_timeout"`
	ShutdownTimeout time.Duration `toml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `toml:"host"`
	Port            int           `toml:"port"`
	Database        string        `toml:"database"`
	User            string        `toml:"user"`
	Password        string        `toml:"password"`
	MaxOpenConns    int           `toml:"max_open_conns"`
	MaxIdleConns    int           `toml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `toml:"conn_max_lifetime"`
}

type PaginationConfig struct {
	DefaultPageSize int `toml:"default_page_size"`
	MaxPageSize     int `toml:"max_page_size"`
}

type CORSConfig struct {
	Origins     []string `toml:"origins"`
	Methods     []string `toml:"methods"`
	Headers     []string `toml:"headers"`
	Credentials bool     `toml:"credentials"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyEnvironmentOverrides(&config)

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func applyEnvironmentOverrides(config *Config) {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("DATABASE_HOST"); host != "" {
		config.Database.Host = host
	}
	if port := os.Getenv("DATABASE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Database.Port = p
		}
	}
	if db := os.Getenv("DATABASE_DATABASE"); db != "" {
		config.Database.Database = db
	}
	if user := os.Getenv("DATABASE_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("DATABASE_PASSWORD"); password != "" {
		config.Database.Password = password
	}

	if defaultPageSize := os.Getenv("PAGINATION_DEFAULT_PAGE_SIZE"); defaultPageSize != "" {
		if ps, err := strconv.Atoi(defaultPageSize); err == nil {
			config.Pagination.DefaultPageSize = ps
		}
	}
	if maxPageSize := os.Getenv("PAGINATION_MAX_PAGE_SIZE"); maxPageSize != "" {
		if ps, err := strconv.Atoi(maxPageSize); err == nil {
			config.Pagination.MaxPageSize = ps
		}
	}

	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		config.CORS.Origins = strings.Split(origins, ",")
	}

	if level := os.Getenv("LOGGING_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOGGING_FORMAT"); format != "" {
		config.Logging.Format = format
	}
}

func validate(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Pagination.DefaultPageSize <= 0 {
		return fmt.Errorf("default page size must be positive")
	}
	if config.Pagination.MaxPageSize <= 0 {
		return fmt.Errorf("max page size must be positive")
	}
	if config.Pagination.DefaultPageSize > config.Pagination.MaxPageSize {
		return fmt.Errorf("default page size cannot exceed max page size")
	}

	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.Host, c.Port, c.Database, c.User, c.Password,
	)
}
```

#### Step 1.7: Main Server Entry Point

**cmd/server/main.go**:
```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JaimeStill/agent-lab/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Application struct {
	config *config.Config
	db     *sql.DB
	logger *slog.Logger
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load("config.toml")
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	db, err := openDB(cfg.Database)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	app := &Application{
		config: cfg,
		db:     db,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      app.routes(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
	}()

	logger.Info("starting server", "addr", srv.Addr)

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	err = <-shutdownError
	if err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func openDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return app.enableCORS(mux)
}

func (app *Application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(app.config.CORS.Origins) > 0 {
			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range app.config.CORS.Origins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if len(app.config.CORS.Methods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", joinStrings(app.config.CORS.Methods))
		}

		if len(app.config.CORS.Headers) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(app.config.CORS.Headers))
		}

		if app.config.CORS.Credentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
```

### Phase 2: Query Engine Implementation

#### Step 2.1: Page Models

**internal/models/page.go**:
```go
package models

type PageOptions struct {
	Page       *int    `json:"page"`
	PageSize   *int    `json:"pageSize"`
	SortBy     *string `json:"sortBy"`
	Descending *bool   `json:"descending"`
	Search     *string `json:"search"`
}

type PageResult[T any] struct {
	Items       []T  `json:"items"`
	Page        int  `json:"page"`
	PageSize    int  `json:"pageSize"`
	TotalCount  int  `json:"totalCount"`
	TotalPages  int  `json:"totalPages"`
	HasPrevious bool `json:"hasPreviousPage"`
	HasNext     bool `json:"hasNextPage"`
}

type SearchRequest[TFilters any] struct {
	Page    PageOptions `json:"page"`
	Filters TFilters    `json:"filters"`
}

type PageValues struct {
	Page       int
	PageSize   int
	TotalCount int
	TotalPages int
}
```

#### Step 2.2: Query Helpers

**internal/query/helpers.go**:
```go
package query

import "reflect"

func isZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0.0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}
```

#### Step 2.3: ProjectionMap

**internal/query/projection_map.go**:
```go
package query

import (
	"fmt"
	"strings"
)

type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
)

type projection struct {
	alias      string
	column     string
	viewName   string
	tableAlias string
}

type join struct {
	table     string
	alias     string
	joinType  JoinType
	condition string
}

type ProjectionMap struct {
	schema        string
	baseTable     string
	baseAlias     string
	projections   []projection
	joins         []join
	currentAlias  string
	columnMapping map[string]projection
}

func NewProjectionMap(schema, table, alias string) *ProjectionMap {
	return &ProjectionMap{
		schema:        schema,
		baseTable:     table,
		baseAlias:     alias,
		currentAlias:  alias,
		projections:   []projection{},
		joins:         []join{},
		columnMapping: make(map[string]projection),
	}
}

func (pm *ProjectionMap) Project(column string, viewName ...string) *ProjectionMap {
	view := column
	if len(viewName) > 0 {
		view = viewName[0]
	}

	proj := projection{
		alias:      pm.currentAlias,
		column:     column,
		viewName:   view,
		tableAlias: pm.currentAlias,
	}

	pm.projections = append(pm.projections, proj)
	pm.columnMapping[view] = proj

	return pm
}

func (pm *ProjectionMap) Join(table, alias string, joinType JoinType, condition string) *ProjectionMap {
	pm.joins = append(pm.joins, join{
		table:     table,
		alias:     alias,
		joinType:  joinType,
		condition: condition,
	})

	pm.currentAlias = alias

	return pm
}

func (pm *ProjectionMap) ResolveColumn(viewProperty string) (string, string) {
	if proj, ok := pm.columnMapping[viewProperty]; ok {
		return proj.tableAlias, proj.column
	}

	return pm.baseAlias, viewProperty
}

func (pm *ProjectionMap) BuildSelectClause() string {
	var columns []string
	for _, proj := range pm.projections {
		if proj.viewName != proj.column {
			columns = append(columns, fmt.Sprintf("%s.%s AS %s", proj.tableAlias, proj.column, proj.viewName))
		} else {
			columns = append(columns, fmt.Sprintf("%s.%s", proj.tableAlias, proj.column))
		}
	}
	return "SELECT " + strings.Join(columns, ", ")
}

func (pm *ProjectionMap) BuildFromClause() string {
	return fmt.Sprintf("FROM %s.%s %s", pm.schema, pm.baseTable, pm.baseAlias)
}

func (pm *ProjectionMap) BuildJoinClauses() string {
	if len(pm.joins) == 0 {
		return ""
	}

	var clauses []string
	for _, j := range pm.joins {
		clauses = append(clauses, fmt.Sprintf("%s %s.%s %s ON %s",
			j.joinType, pm.schema, j.table, j.alias, j.condition))
	}
	return strings.Join(clauses, "\n")
}
```

#### Step 2.4: QueryBuilder

**internal/query/query_builder.go**:
```go
package query

import (
	"fmt"
	"strings"
)

type QueryBuilder struct {
	projection   *ProjectionMap
	whereClauses []string
	args         []interface{}
	orderBy      string
	descending   bool
	defaultSort  string
}

func NewBuilder(projection *ProjectionMap, defaultSort string) *QueryBuilder {
	return &QueryBuilder{
		projection:   projection,
		whereClauses: []string{},
		args:         []interface{}{},
		defaultSort:  defaultSort,
	}
}

func (qb *QueryBuilder) WhereEquals(viewProperty string, value interface{}) *QueryBuilder {
	if value != nil && !isZeroValue(value) {
		alias, column := qb.projection.ResolveColumn(viewProperty)
		qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s.%s = $%d", alias, column, len(qb.args)+1))
		qb.args = append(qb.args, value)
	}
	return qb
}

func (qb *QueryBuilder) WhereContains(viewProperty string, value string) *QueryBuilder {
	if value != "" {
		alias, column := qb.projection.ResolveColumn(viewProperty)
		qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("%s.%s LIKE $%d", alias, column, len(qb.args)+1))
		qb.args = append(qb.args, "%"+value+"%")
	}
	return qb
}

func (qb *QueryBuilder) WhereSearch(searchValue string, viewProperties ...string) *QueryBuilder {
	if searchValue != "" && len(viewProperties) > 0 {
		var searchClauses []string
		searchPattern := "%" + searchValue + "%"

		for range viewProperties {
			qb.args = append(qb.args, searchPattern)
		}

		for i, prop := range viewProperties {
			alias, column := qb.projection.ResolveColumn(prop)
			searchClauses = append(searchClauses, fmt.Sprintf("%s.%s LIKE $%d", alias, column, len(qb.args)-len(viewProperties)+i+1))
		}

		qb.whereClauses = append(qb.whereClauses, fmt.Sprintf("(%s)", strings.Join(searchClauses, " OR ")))
	}
	return qb
}

func (qb *QueryBuilder) OrderBy(viewProperty *string, descending *bool) *QueryBuilder {
	if viewProperty != nil && *viewProperty != "" {
		qb.orderBy = *viewProperty
	} else {
		qb.orderBy = qb.defaultSort
	}

	if descending != nil {
		qb.descending = *descending
	}

	return qb
}

func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
	sql := "SELECT COUNT(*) " + qb.projection.BuildFromClause()

	joinClauses := qb.projection.BuildJoinClauses()
	if joinClauses != "" {
		sql += "\n" + joinClauses
	}

	if len(qb.whereClauses) > 0 {
		sql += "\nWHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	return sql, qb.args
}

func (qb *QueryBuilder) BuildPage(page, pageSize int) (string, []interface{}) {
	sql := qb.projection.BuildSelectClause() + "\n" + qb.projection.BuildFromClause()

	joinClauses := qb.projection.BuildJoinClauses()
	if joinClauses != "" {
		sql += "\n" + joinClauses
	}

	if len(qb.whereClauses) > 0 {
		sql += "\nWHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	alias, column := qb.projection.ResolveColumn(qb.orderBy)
	direction := "ASC"
	if qb.descending {
		direction = "DESC"
	}
	sql += fmt.Sprintf("\nORDER BY %s.%s %s", alias, column, direction)

	offset := (page - 1) * pageSize
	sql += fmt.Sprintf("\nOFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, pageSize)

	return sql, qb.args
}

func (qb *QueryBuilder) BuildSingle() (string, []interface{}) {
	sql := qb.projection.BuildSelectClause() + "\n" + qb.projection.BuildFromClause()

	joinClauses := qb.projection.BuildJoinClauses()
	if joinClauses != "" {
		sql += "\n" + joinClauses
	}

	if len(qb.whereClauses) > 0 {
		sql += "\nWHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	return sql, qb.args
}
```

#### Step 2.5: Pagination Service

**internal/services/pagination.go**:
```go
package services

import (
	"math"

	"github.com/JaimeStill/agent-lab/internal/config"
	"github.com/JaimeStill/agent-lab/internal/models"
)

type PaginationService struct {
	config config.PaginationConfig
}

func NewPaginationService(config config.PaginationConfig) *PaginationService {
	return &PaginationService{
		config: config,
	}
}

func (s *PaginationService) CalculatePagination(opts models.PageOptions, totalCount int) models.PageValues {
	pageSize := s.config.DefaultPageSize
	if opts.PageSize != nil && *opts.PageSize > 0 {
		if *opts.PageSize > s.config.MaxPageSize {
			pageSize = s.config.MaxPageSize
		} else {
			pageSize = *opts.PageSize
		}
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	page := 1
	if opts.Page != nil {
		page = *opts.Page
		if page < 1 {
			page = 1
		} else if page > totalPages && totalPages > 0 {
			page = totalPages
		}
	}

	return models.PageValues{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
```

### Phase 3: Provider Configuration Management

#### Step 3.1: Provider Model

**internal/models/provider.go**:
```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Provider struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Config    []byte    `json:"config"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ProviderFilters struct {
	Name *string `json:"name"`
}

type CreateProviderRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}

type UpdateProviderRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}
```

#### Step 3.2: Service Errors

**internal/services/errors.go**:
```go
package services

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidConfig = errors.New("invalid configuration")
)
```

#### Step 3.3: Provider Service

**internal/services/provider.go**:
```go
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/JaimeStill/agent-lab/internal/models"
	"github.com/JaimeStill/agent-lab/internal/query"
	"github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/go-agents/pkg/providers"
	"github.com/google/uuid"
)

var providerProjection = query.NewProjectionMap("public", "providers", "p").
	Project("id", "Id").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

type ProviderService struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination *PaginationService
}

func NewProviderService(db *sql.DB, logger *slog.Logger, pagination *PaginationService) *ProviderService {
	return &ProviderService{
		db:         db,
		logger:     logger.With("service", "provider"),
		pagination: pagination,
	}
}

func (s *ProviderService) FromRequest(name string, configJSON []byte) (*models.Provider, error) {
	var providerConfig config.ProviderConfig
	if err := json.Unmarshal(configJSON, &providerConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	_, err := providers.New(&providerConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return &models.Provider{
		ID:        uuid.New(),
		Name:      name,
		Config:    configJSON,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (s *ProviderService) Create(ctx context.Context, name string, configJSON []byte) (*models.Provider, error) {
	provider, err := s.FromRequest(name, configJSON)
	if err != nil {
		return nil, err
	}

	sqlQuery := `
		INSERT INTO providers (id, name, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = s.db.ExecContext(ctx, sqlQuery, provider.ID, provider.Name, provider.Config, provider.CreatedAt, provider.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	s.logger.Info("provider created", "id", provider.ID, "name", provider.Name)

	return provider, nil
}

func (s *ProviderService) GetByID(ctx context.Context, id uuid.UUID) (*models.Provider, error) {
	qb := query.NewBuilder(providerProjection, "Name").
		WhereEquals("Id", id)

	sqlQuery, args := qb.BuildSingle()

	var provider models.Provider
	err := s.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Config,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return &provider, nil
}

func (s *ProviderService) Update(ctx context.Context, id uuid.UUID, name string, configJSON []byte) (*models.Provider, error) {
	provider, err := s.FromRequest(name, configJSON)
	if err != nil {
		return nil, err
	}

	provider.ID = id
	provider.UpdatedAt = time.Now()

	sqlQuery := `
		UPDATE providers
		SET name = $2, config = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, sqlQuery, provider.ID, provider.Name, provider.Config, provider.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	s.logger.Info("provider updated", "id", provider.ID, "name", provider.Name)

	return provider, nil
}

func (s *ProviderService) Delete(ctx context.Context, id uuid.UUID) error {
	sqlQuery := `DELETE FROM providers WHERE id = $1`

	result, err := s.db.ExecContext(ctx, sqlQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	s.logger.Info("provider deleted", "id", id)

	return nil
}

func (s *ProviderService) Search(ctx context.Context, page models.PageOptions, filters models.ProviderFilters) (*models.PageResult[models.Provider], error) {
	qb := query.NewBuilder(providerProjection, "Name").
		WhereContains("Name", derefString(filters.Name)).
		WhereSearch(derefString(page.Search), "Name").
		OrderBy(page.SortBy, page.Descending)

	countSQL, countArgs := qb.BuildCount()
	var totalCount int
	err := s.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count providers: %w", err)
	}

	pagination := s.pagination.CalculatePagination(page, totalCount)

	pageSQL, pageArgs := qb.BuildPage(pagination.Page, pagination.PageSize)
	rows, err := s.db.QueryContext(ctx, pageSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query providers: %w", err)
	}
	defer rows.Close()

	var providers []models.Provider
	for rows.Next() {
		var provider models.Provider
		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Config,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, provider)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating providers: %w", err)
	}

	return &models.PageResult[models.Provider]{
		Items:       providers,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalCount:  pagination.TotalCount,
		TotalPages:  pagination.TotalPages,
		HasPrevious: pagination.Page > 1,
		HasNext:     pagination.Page < pagination.TotalPages,
	}, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

#### Step 3.4: Provider Handlers

**internal/handlers/provider.go**:
```go
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/models"
	"github.com/JaimeStill/agent-lab/internal/services"
	"github.com/google/uuid"
)

type ProviderHandler struct {
	service *services.ProviderService
	logger  *slog.Logger
}

func NewProviderHandler(service *services.ProviderService, logger *slog.Logger) *ProviderHandler {
	return &ProviderHandler{
		service: service,
		logger:  logger.With("handler", "provider"),
	}
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.CreateProviderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}

	provider, err := h.service.Create(r.Context(), req.Name, req.Config)
	if err != nil {
		if errors.Is(err, services.ErrInvalidConfig) {
			h.errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("failed to create provider", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusCreated, provider)
}

func (h *ProviderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	provider, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "provider not found")
			return
		}
		h.logger.Error("failed to get provider", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, provider)
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.UpdateProviderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}

	provider, err := h.service.Update(r.Context(), id, req.Name, req.Config)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "provider not found")
			return
		}
		if errors.Is(err, services.ErrInvalidConfig) {
			h.errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("failed to update provider", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, provider)
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "provider not found")
			return
		}
		h.logger.Error("failed to delete provider", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProviderHandler) Search(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.SearchRequest[models.ProviderFilters]
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	result, err := h.service.Search(r.Context(), req.Page, req.Filters)
	if err != nil {
		h.logger.Error("failed to search providers", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

func (h *ProviderHandler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *ProviderHandler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}
```

#### Step 3.5: Update Routes

**cmd/server/main.go** - Update `routes()` method:
```go
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	paginationService := services.NewPaginationService(app.config.Pagination)

	providerService := services.NewProviderService(app.db, app.logger, paginationService)
	providerHandler := handlers.NewProviderHandler(providerService, app.logger)

	mux.HandleFunc("POST /api/providers", providerHandler.Create)
	mux.HandleFunc("GET /api/providers/{id}", providerHandler.GetByID)
	mux.HandleFunc("PUT /api/providers/{id}", providerHandler.Update)
	mux.HandleFunc("DELETE /api/providers/{id}", providerHandler.Delete)
	mux.HandleFunc("POST /api/providers/search", providerHandler.Search)

	return app.enableCORS(mux)
}
```

### Phase 4: Agent Configuration Management

#### Step 4.1: Agent Model

**internal/models/agent.go**:
```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Config    []byte    `json:"config"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AgentFilters struct {
	Name *string `json:"name"`
}

type CreateAgentRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}

type UpdateAgentRequest struct {
	Name   string `json:"name"`
	Config []byte `json:"config"`
}
```

#### Step 4.2: Agent Service

**internal/services/agent.go**:
```go
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/JaimeStill/agent-lab/internal/models"
	"github.com/JaimeStill/agent-lab/internal/query"
	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/go-agents/pkg/config"
	"github.com/google/uuid"
)

var agentProjection = query.NewProjectionMap("public", "agents", "a").
	Project("id", "Id").
	Project("name", "Name").
	Project("config", "Config").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

type AgentService struct {
	db         *sql.DB
	logger     *slog.Logger
	pagination *PaginationService
}

func NewAgentService(db *sql.DB, logger *slog.Logger, pagination *PaginationService) *AgentService {
	return &AgentService{
		db:         db,
		logger:     logger.With("service", "agent"),
		pagination: pagination,
	}
}

func (s *AgentService) FromRequest(name string, configJSON []byte) (*models.Agent, error) {
	var agentConfig config.AgentConfig
	if err := json.Unmarshal(configJSON, &agentConfig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	_, err := agent.New(&agentConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return &models.Agent{
		ID:        uuid.New(),
		Name:      name,
		Config:    configJSON,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (s *AgentService) Create(ctx context.Context, name string, configJSON []byte) (*models.Agent, error) {
	agentModel, err := s.FromRequest(name, configJSON)
	if err != nil {
		return nil, err
	}

	sqlQuery := `
		INSERT INTO agents (id, name, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = s.db.ExecContext(ctx, sqlQuery, agentModel.ID, agentModel.Name, agentModel.Config, agentModel.CreatedAt, agentModel.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	s.logger.Info("agent created", "id", agentModel.ID, "name", agentModel.Name)

	return agentModel, nil
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	qb := query.NewBuilder(agentProjection, "Name").
		WhereEquals("Id", id)

	sqlQuery, args := qb.BuildSingle()

	var agentModel models.Agent
	err := s.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&agentModel.ID,
		&agentModel.Name,
		&agentModel.Config,
		&agentModel.CreatedAt,
		&agentModel.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return &agentModel, nil
}

func (s *AgentService) Update(ctx context.Context, id uuid.UUID, name string, configJSON []byte) (*models.Agent, error) {
	agentModel, err := s.FromRequest(name, configJSON)
	if err != nil {
		return nil, err
	}

	agentModel.ID = id
	agentModel.UpdatedAt = time.Now()

	sqlQuery := `
		UPDATE agents
		SET name = $2, config = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, sqlQuery, agentModel.ID, agentModel.Name, agentModel.Config, agentModel.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	s.logger.Info("agent updated", "id", agentModel.ID, "name", agentModel.Name)

	return agentModel, nil
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	sqlQuery := `DELETE FROM agents WHERE id = $1`

	result, err := s.db.ExecContext(ctx, sqlQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	s.logger.Info("agent deleted", "id", id)

	return nil
}

func (s *AgentService) Search(ctx context.Context, page models.PageOptions, filters models.AgentFilters) (*models.PageResult[models.Agent], error) {
	qb := query.NewBuilder(agentProjection, "Name").
		WhereContains("Name", derefString(filters.Name)).
		WhereSearch(derefString(page.Search), "Name").
		OrderBy(page.SortBy, page.Descending)

	countSQL, countArgs := qb.BuildCount()
	var totalCount int
	err := s.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count agents: %w", err)
	}

	pagination := s.pagination.CalculatePagination(page, totalCount)

	pageSQL, pageArgs := qb.BuildPage(pagination.Page, pagination.PageSize)
	rows, err := s.db.QueryContext(ctx, pageSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agentModel models.Agent
		err := rows.Scan(
			&agentModel.ID,
			&agentModel.Name,
			&agentModel.Config,
			&agentModel.CreatedAt,
			&agentModel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agentModel)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return &models.PageResult[models.Agent]{
		Items:       agents,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalCount:  pagination.TotalCount,
		TotalPages:  pagination.TotalPages,
		HasPrevious: pagination.Page > 1,
		HasNext:     pagination.Page < pagination.TotalPages,
	}, nil
}

func (s *AgentService) LoadAgentConfig(ctx context.Context, id uuid.UUID) (*config.AgentConfig, error) {
	agentModel, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var agentConfig config.AgentConfig
	if err := json.Unmarshal(agentModel.Config, &agentConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent config: %w", err)
	}

	return &agentConfig, nil
}

func (s *AgentService) CreateAgent(ctx context.Context, id uuid.UUID, token string) (agent.Agent, error) {
	agentConfig, err := s.LoadAgentConfig(ctx, id)
	if err != nil {
		return nil, err
	}

	if token != "" {
		if agentConfig.Client.Provider.Options == nil {
			agentConfig.Client.Provider.Options = make(map[string]any)
		}

		if authType, ok := agentConfig.Client.Provider.Options["auth_type"].(string); ok && authType == "bearer" {
			agentConfig.Client.Provider.Options["token"] = token
		}
	}

	return agent.New(agentConfig)
}
```

#### Step 4.3: Agent Handlers

**internal/handlers/agent.go**:
```go
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/models"
	"github.com/JaimeStill/agent-lab/internal/services"
	"github.com/google/uuid"
)

type AgentHandler struct {
	service *services.AgentService
	logger  *slog.Logger
}

func NewAgentHandler(service *services.AgentService, logger *slog.Logger) *AgentHandler {
	return &AgentHandler{
		service: service,
		logger:  logger.With("handler", "agent"),
	}
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.CreateAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}

	agent, err := h.service.Create(r.Context(), req.Name, req.Config)
	if err != nil {
		if errors.Is(err, services.ErrInvalidConfig) {
			h.errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusCreated, agent)
}

func (h *AgentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	agent, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to get agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, agent)
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.UpdateAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Name == "" {
		h.errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}

	agent, err := h.service.Update(r.Context(), id, req.Name, req.Config)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		if errors.Is(err, services.ErrInvalidConfig) {
			h.errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("failed to update agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, agent)
}

func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to delete agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) Search(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.SearchRequest[models.AgentFilters]
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	result, err := h.service.Search(r.Context(), req.Page, req.Filters)
	if err != nil {
		h.logger.Error("failed to search agents", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

func (h *AgentHandler) extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

func (h *AgentHandler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *AgentHandler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}
```

#### Step 4.4: Update Routes

**cmd/server/main.go** - Update `routes()` method to add agent endpoints:
```go
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	paginationService := services.NewPaginationService(app.config.Pagination)

	providerService := services.NewProviderService(app.db, app.logger, paginationService)
	providerHandler := handlers.NewProviderHandler(providerService, app.logger)

	mux.HandleFunc("POST /api/providers", providerHandler.Create)
	mux.HandleFunc("GET /api/providers/{id}", providerHandler.GetByID)
	mux.HandleFunc("PUT /api/providers/{id}", providerHandler.Update)
	mux.HandleFunc("DELETE /api/providers/{id}", providerHandler.Delete)
	mux.HandleFunc("POST /api/providers/search", providerHandler.Search)

	agentService := services.NewAgentService(app.db, app.logger, paginationService)
	agentHandler := handlers.NewAgentHandler(agentService, app.logger)

	mux.HandleFunc("POST /api/agents", agentHandler.Create)
	mux.HandleFunc("GET /api/agents/{id}", agentHandler.GetByID)
	mux.HandleFunc("PUT /api/agents/{id}", agentHandler.Update)
	mux.HandleFunc("DELETE /api/agents/{id}", agentHandler.Delete)
	mux.HandleFunc("POST /api/agents/search", agentHandler.Search)

	return app.enableCORS(mux)
}
```

### Phase 5: Protocol Execution Endpoints

#### Step 5.1: Protocol Request Models

**internal/models/protocol.go**:
```go
package models

import "github.com/JaimeStill/go-agents/pkg/agent"

type ChatRequest struct {
	Prompt  string         `json:"prompt"`
	Options map[string]any `json:"options,omitempty"`
}

type VisionRequest struct {
	Prompt  string         `json:"prompt"`
	Images  []string       `json:"images"`
	Options map[string]any `json:"options,omitempty"`
}

type ToolsRequest struct {
	Prompt  string         `json:"prompt"`
	Tools   []agent.Tool   `json:"tools"`
	Options map[string]any `json:"options,omitempty"`
}

type EmbeddingsRequest struct {
	Input any `json:"input"`
}
```

#### Step 5.2: Protocol Handlers

Add to **internal/handlers/agent.go**:
```go
func (h *AgentHandler) Chat(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Prompt == "" {
		h.errorResponse(w, http.StatusBadRequest, "prompt is required")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response, err := agent.Chat(r.Context(), req.Prompt, req.Options)
	if err != nil {
		h.logger.Error("chat failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "chat execution failed")
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}

func (h *AgentHandler) ChatStream(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Prompt == "" {
		h.errorResponse(w, http.StatusBadRequest, "prompt is required")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	stream, err := agent.ChatStream(r.Context(), req.Prompt, req.Options)
	if err != nil {
		h.logger.Error("chat stream failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "chat stream execution failed")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.errorResponse(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	for chunk := range stream {
		if chunk.Error != nil {
			h.logger.Error("stream error", "error", chunk.Error)
			break
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			h.logger.Error("failed to marshal chunk", "error", err)
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (h *AgentHandler) Vision(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.VisionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Prompt == "" {
		h.errorResponse(w, http.StatusBadRequest, "prompt is required")
		return
	}

	if len(req.Images) == 0 {
		h.errorResponse(w, http.StatusBadRequest, "images are required")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response, err := agent.Vision(r.Context(), req.Prompt, req.Images, req.Options)
	if err != nil {
		h.logger.Error("vision failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "vision execution failed")
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}

func (h *AgentHandler) VisionStream(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.VisionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Prompt == "" {
		h.errorResponse(w, http.StatusBadRequest, "prompt is required")
		return
	}

	if len(req.Images) == 0 {
		h.errorResponse(w, http.StatusBadRequest, "images are required")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	stream, err := agent.VisionStream(r.Context(), req.Prompt, req.Images, req.Options)
	if err != nil {
		h.logger.Error("vision stream failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "vision stream execution failed")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.errorResponse(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	for chunk := range stream {
		if chunk.Error != nil {
			h.logger.Error("stream error", "error", chunk.Error)
			break
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			h.logger.Error("failed to marshal chunk", "error", err)
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (h *AgentHandler) Tools(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.ToolsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Prompt == "" {
		h.errorResponse(w, http.StatusBadRequest, "prompt is required")
		return
	}

	if len(req.Tools) == 0 {
		h.errorResponse(w, http.StatusBadRequest, "tools are required")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response, err := agent.Tools(r.Context(), req.Prompt, req.Tools, req.Options)
	if err != nil {
		h.logger.Error("tools failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "tools execution failed")
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}

func (h *AgentHandler) Embed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid agent id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req models.EmbeddingsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request format")
		return
	}

	if req.Input == nil {
		h.errorResponse(w, http.StatusBadRequest, "input is required")
		return
	}

	inputStr, ok := req.Input.(string)
	if !ok {
		h.errorResponse(w, http.StatusBadRequest, "input must be a string")
		return
	}

	token := h.extractBearerToken(r.Header.Get("Authorization"))

	agent, err := h.service.CreateAgent(r.Context(), id, token)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			h.errorResponse(w, http.StatusNotFound, "agent not found")
			return
		}
		h.logger.Error("failed to create agent", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response, err := agent.Embed(r.Context(), inputStr)
	if err != nil {
		h.logger.Error("embed failed", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "embed execution failed")
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}
```

Add import for `fmt` at top of file:
```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/models"
	"github.com/JaimeStill/agent-lab/internal/services"
	"github.com/google/uuid"
)
```

#### Step 5.3: Update Routes

**cmd/server/main.go** - Update `routes()` method to add protocol endpoints:
```go
func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	paginationService := services.NewPaginationService(app.config.Pagination)

	providerService := services.NewProviderService(app.db, app.logger, paginationService)
	providerHandler := handlers.NewProviderHandler(providerService, app.logger)

	mux.HandleFunc("POST /api/providers", providerHandler.Create)
	mux.HandleFunc("GET /api/providers/{id}", providerHandler.GetByID)
	mux.HandleFunc("PUT /api/providers/{id}", providerHandler.Update)
	mux.HandleFunc("DELETE /api/providers/{id}", providerHandler.Delete)
	mux.HandleFunc("POST /api/providers/search", providerHandler.Search)

	agentService := services.NewAgentService(app.db, app.logger, paginationService)
	agentHandler := handlers.NewAgentHandler(agentService, app.logger)

	mux.HandleFunc("POST /api/agents", agentHandler.Create)
	mux.HandleFunc("GET /api/agents/{id}", agentHandler.GetByID)
	mux.HandleFunc("PUT /api/agents/{id}", agentHandler.Update)
	mux.HandleFunc("DELETE /api/agents/{id}", agentHandler.Delete)
	mux.HandleFunc("POST /api/agents/search", agentHandler.Search)

	mux.HandleFunc("POST /api/agents/{id}/chat", agentHandler.Chat)
	mux.HandleFunc("POST /api/agents/{id}/chat/stream", agentHandler.ChatStream)
	mux.HandleFunc("POST /api/agents/{id}/vision", agentHandler.Vision)
	mux.HandleFunc("POST /api/agents/{id}/vision/stream", agentHandler.VisionStream)
	mux.HandleFunc("POST /api/agents/{id}/tools", agentHandler.Tools)
	mux.HandleFunc("POST /api/agents/{id}/embed", agentHandler.Embed)

	return app.enableCORS(mux)
}
```

### Phase 6: OpenAPI & Scalar Integration

This phase will be completed after validation and testing infrastructure is established. It consists of:

1. Download Scalar standalone bundle to `web/scalar/`
2. Create `api/openapi.yaml` with complete API specification
3. Add endpoint in `routes()` to serve Scalar UI at `/api/docs`
4. Add endpoint to serve OpenAPI spec at `/api/openapi.yaml`

The AI will create and maintain the OpenAPI specification as part of the documentation phase after validation.

## Next Steps

1. **Developer Execution**: Execute the implementation guide step-by-step
2. **Validation Phase**: AI reviews, adds tests, validates coverage
3. **Documentation Phase**: AI adds godoc comments
4. **OpenAPI Creation**: AI creates api/openapi.yaml and integrates Scalar UI
5. **Session Closeout**: Archive guide, update project docs
