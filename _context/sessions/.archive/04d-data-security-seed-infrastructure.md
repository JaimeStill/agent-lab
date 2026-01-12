# Session 4d: Data Security and Seed Infrastructure

## Problem Context

Session 4c revealed a security issue: the cloud provider token is being captured in workflow state and stored in the database. This includes:
- `checkpoints.state_data` (full state serialization)
- `stages.input_snapshot` / `stages.output_snapshot` (observer events)
- `runs.result` (final state data)
- SSE events (streamed to clients)

The root cause is architectural: `state.State.Data` has no separation between persistable data and secrets. All data in `Data` is serialized and observable.

Additionally, we need infrastructure for profile experimentation without constantly modifying code.

## Architecture Approach

### Phase 1: go-agents-orchestration - Add Secrets Support

Add a dedicated `Secrets` field to `state.State` that is explicitly excluded from persistence and observability:

```go
type State struct {
    Data           map[string]any         `json:"data"`
    Secrets        map[string]any         `json:"-"`   // Never persisted/observed
    Observer       observability.Observer `json:"-"`
    RunID          string                 `json:"run_id"`
    CheckpointNode string                 `json:"checkpoint_node"`
    Timestamp      time.Time              `json:"timestamp"`
}
```

**Why this works**:
- `json:"-"` excludes Secrets from checkpoint serialization
- Observer snapshots use `maps.Clone(state.Data)` at graph.go:419, 433 - Secrets naturally excluded
- Explicit API makes intent clear (SetSecret vs Set)

### Phase 2: agent-lab Integration

Update agent-lab to use the new Secrets API:
- `executor.go`: Use `SetSecret("token", token)` instead of `Set("token", token)`
- `profile.go`: Use `GetSecret("token")` to retrieve token

### Phase 3: cmd/seed Infrastructure

Create a new CLI tool for seeding data, following the `cmd/migrate` pattern:
- Seeder interface for extensibility
- Transactional execution (all-or-nothing)
- Save semantics (creates or updates - safe to run multiple times)
- Embedded seed files with optional external file override

---

## Phase 1: go-agents-orchestration Changes

All changes are in `/home/jaime/code/go-agents-orchestration/`.

### Step 1.1: Update state.State struct

**File**: `pkg/state/state.go`

Add the Secrets field to the State struct:

```go
type State struct {
	Data           map[string]any         `json:"data"`
	Secrets        map[string]any         `json:"-"`
	Observer       observability.Observer `json:"-"`
	RunID          string                 `json:"run_id"`
	CheckpointNode string                 `json:"checkpoint_node"`
	Timestamp      time.Time              `json:"timestamp"`
}
```

### Step 1.2: Update New() function

**File**: `pkg/state/state.go`

Initialize the Secrets map in the New function:

```go
func New(observer observability.Observer) State {
	if observer == nil {
		observer = observability.NoOpObserver{}
	}

	s := State{
		Data:      make(map[string]any),
		Secrets:   make(map[string]any),
		Observer:  observer,
		RunID:     uuid.New().String(),
		Timestamp: time.Now(),
	}

	observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateCreate,
		Timestamp: s.Timestamp,
		Source:    "state",
		Data:      map[string]any{},
	})

	return s
}
```

### Step 1.3: Update Clone() method

**File**: `pkg/state/state.go`

Clone the Secrets map along with Data:

```go
func (s State) Clone() State {
	newState := State{
		Data:           maps.Clone(s.Data),
		Secrets:        maps.Clone(s.Secrets),
		Observer:       s.Observer,
		RunID:          s.RunID,
		CheckpointNode: s.CheckpointNode,
		Timestamp:      s.Timestamp,
	}

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateClone,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(newState.Data)},
	})

	return newState
}
```

### Step 1.4: Add secret manipulation methods

**File**: `pkg/state/state.go`

Add these methods after the existing Set method:

```go
func (s State) GetSecret(key string) (any, bool) {
	val, exists := s.Secrets[key]
	return val, exists
}

func (s State) SetSecret(key string, value any) State {
	state := s.Clone()
	state.Secrets[key] = value
	return state
}

func (s State) DeleteSecret(key string) State {
	state := s.Clone()
	delete(state.Secrets, key)
	return state
}
```

### Step 1.5: Update CHANGELOG.md

**File**: `CHANGELOG.md`

Add entry at the top:

```markdown
## [v0.3.2] - 2025-XX-XX

### Added

- `pkg/state` - Secrets field for sensitive data that should never be persisted or observed

  State now has a `Secrets` field (`map[string]any`) with `json:"-"` tag that is excluded from JSON serialization and observer snapshots. New methods `SetSecret`, `GetSecret`, and `DeleteSecret` provide explicit API for managing sensitive data like authentication tokens.
```

### Step 1.6: Release new version

After implementing and testing:
1. Commit changes
2. Tag as v0.3.2
3. Push tag to trigger release

---

## Phase 2: agent-lab Integration

All changes are in `/home/jaime/code/agent-lab/`.

### Step 2.1: Update go.mod dependency

**File**: `go.mod`

Update the go-agents-orchestration dependency to the new version:

```
require (
    github.com/JaimeStill/go-agents-orchestration v0.3.2
)
```

Run `go mod tidy` after updating.

### Step 2.2: Update executor.go - Execute method

**File**: `internal/workflows/executor.go`

Change the token handling in the Execute method (around line 106-108):

```go
initialState.RunID = run.ID.String()
if token != "" {
	initialState = initialState.SetSecret("token", token)
}
```

### Step 2.3: Update executor.go - executeStreamAsync method

**File**: `internal/workflows/executor.go`

Change the token handling in executeStreamAsync (around line 254-256):

```go
initialState.RunID = runID.String()
if token != "" {
	initialState = initialState.SetSecret("token", token)
}
```

### Step 2.4: Update profile.go - ExtractAgentParams function

**File**: `internal/workflows/profile.go`

Update ExtractAgentParams to use GetSecret:

```go
func ExtractAgentParams(s state.State, stage *profiles.ProfileStage) (uuid.UUID, string, error) {
	var agentID uuid.UUID
	var token string

	if stage != nil && stage.AgentID != nil {
		agentID = *stage.AgentID
	} else {
		agentIDVal, ok := s.Get("agent_id")
		if !ok {
			return uuid.Nil, "", fmt.Errorf("agent_id not found in state or profile stage")
		}
		var err error
		agentID, err = uuid.Parse(agentIDVal.(string))
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("invalid agent_id: %w", err)
		}
	}

	if tokenVal, ok := s.GetSecret("token"); ok {
		token = tokenVal.(string)
	}

	return agentID, token, nil
}
```

---

## Phase 3: cmd/seed Infrastructure

All new files in `/home/jaime/code/agent-lab/cmd/seed/`.

### Step 3.1: Create directory structure

```
cmd/seed/
├── main.go
├── seeder.go
├── seeds.go
├── profiles.go
└── seeds/
    └── classify_profiles.json
```

### Step 3.2: Create seeder.go

**File**: `cmd/seed/seeder.go`

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
)

type Seeder interface {
	Name() string
	Description() string
	Seed(ctx context.Context, tx *sql.Tx) error
}

var seeders = map[string]Seeder{}

func registerSeeder(s Seeder) {
	seeders[s.Name()] = s
}

func getSeeder(name string) (Seeder, bool) {
	s, ok := seeders[name]
	return s, ok
}

func listSeeders() []Seeder {
	result := make([]Seeder, 0, len(seeders))
	for _, s := range seeders {
		result = append(result, s)
	}
	return result
}

func runSeeder(ctx context.Context, db *sql.DB, name string) error {
	seeder, ok := getSeeder(name)
	if !ok {
		return fmt.Errorf("seeder not found: %s", name)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := seeder.Seed(ctx, tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("seed %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func runAllSeeders(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	for name, seeder := range seeders {
		if err := seeder.Seed(ctx, tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("seed %s: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
```

### Step 3.3: Create seeds.go

**File**: `cmd/seed/seeds.go`

Shared embedded filesystem for all seeders to reference their default seed files:

```go
package main

import "embed"

//go:embed seeds/*.json
var seedFiles embed.FS
```

### Step 3.4: Create profiles.go

**File**: `cmd/seed/profiles.go`

Uses existing `ProfileWithStages` type from `internal/profiles` to avoid redundant type definitions. During JSON unmarshaling, auto-generated fields (ID, timestamps, ProfileID) get zero values which are ignored - we generate new ones during save.

```go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/google/uuid"
)

func init() {
	registerSeeder(&ProfileSeeder{})
}

type ProfileSeedData struct {
	Profiles []profiles.ProfileWithStages `json:"profiles"`
}

type ProfileSeeder struct {
	file string
}

func (s *ProfileSeeder) Name() string {
	return "profiles"
}

func (s *ProfileSeeder) Description() string {
	return "Seeds workflow profiles and their stage configurations"
}

func (s *ProfileSeeder) SetFile(path string) {
	s.file = path
}

func (s *ProfileSeeder) Seed(ctx context.Context, tx *sql.Tx) error {
	data, err := s.loadSeedData()
	if err != nil {
		return err
	}

	for _, p := range data.Profiles {
		profileID, err := s.saveProfile(ctx, tx, p.Profile)
		if err != nil {
			return fmt.Errorf("save profile %s/%s: %w", p.WorkflowName, p.Name, err)
		}

		for _, stage := range p.Stages {
			if err := s.saveStage(ctx, tx, profileID, stage); err != nil {
				return fmt.Errorf("save stage %s for profile %s: %w", stage.StageName, p.Name, err)
			}
		}
	}

	return nil
}

func (s *ProfileSeeder) loadSeedData() (*ProfileSeedData, error) {
	var content []byte
	var err error

	if s.file != "" {
		content, err = os.ReadFile(s.file)
		if err != nil {
			return nil, fmt.Errorf("read seed file: %w", err)
		}
	} else {
		content, err = seedFiles.ReadFile("seeds/classify_profiles.json")
		if err != nil {
			return nil, fmt.Errorf("read embedded seed file: %w", err)
		}
	}

	var data ProfileSeedData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("parse seed data: %w", err)
	}

	return &data, nil
}

func (s *ProfileSeeder) saveProfile(ctx context.Context, tx *sql.Tx, p profiles.Profile) (uuid.UUID, error) {
	const query = `
		INSERT INTO profiles (id, workflow_name, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (workflow_name, name) DO UPDATE SET
			description = EXCLUDED.description,
			updated_at = NOW()
		RETURNING id
	`

	id := uuid.New()
	var returnedID uuid.UUID
	err := tx.QueryRowContext(ctx, query, id, p.WorkflowName, p.Name, p.Description).Scan(&returnedID)
	if err != nil {
		return uuid.Nil, err
	}

	return returnedID, nil
}

func (s *ProfileSeeder) saveStage(ctx context.Context, tx *sql.Tx, profileID uuid.UUID, stage profiles.ProfileStage) error {
	const query = `
		INSERT INTO profile_stages (profile_id, stage_name, agent_id, system_prompt, options)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (profile_id, stage_name) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			system_prompt = EXCLUDED.system_prompt,
			options = EXCLUDED.options
	`

	_, err := tx.ExecContext(ctx, query, profileID, stage.StageName, stage.AgentID, stage.SystemPrompt, stage.Options)
	return err
}
```

### Step 3.5: Create main.go

**File**: `cmd/seed/main.go`

```go
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const EnvDatabaseDSN = "DATABASE_DSN"

func main() {
	var (
		dsn      = flag.String("dsn", "", "Database connection string")
		all      = flag.Bool("all", false, "Run all seeders")
		profiles = flag.Bool("profiles", false, "Seed profiles")
		file     = flag.String("file", "", "External seed file (overrides embedded)")
		list     = flag.Bool("list", false, "List available seeders")
	)
	flag.Parse()

	if *list {
		fmt.Println("Available seeders:")
		for _, s := range listSeeders() {
			fmt.Printf("  - %s: %s\n", s.Name(), s.Description())
		}
		return
	}

	if *dsn == "" {
		*dsn = os.Getenv(EnvDatabaseDSN)
	}
	if *dsn == "" {
		log.Fatalf("database connection string required: use -dsn flag or %s env var", EnvDatabaseDSN)
	}

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	ctx := context.Background()

	switch {
	case *all:
		if err := runAllSeeders(ctx, db); err != nil {
			log.Fatalf("seeding failed: %v", err)
		}
		fmt.Println("all seeders completed successfully")

	case *profiles:
		if *file != "" {
			if seeder, ok := getSeeder("profiles"); ok {
				seeder.(*ProfileSeeder).SetFile(*file)
			}
		}
		if err := runSeeder(ctx, db, "profiles"); err != nil {
			log.Fatalf("seeding failed: %v", err)
		}
		fmt.Println("profiles seeded successfully")

	default:
		fmt.Println("usage: seed -dsn <connection-string> [-all|-profiles] [-file <path>] [-list]")
		flag.PrintDefaults()
	}
}
```

### Step 3.6: Create seed data file

**File**: `cmd/seed/seeds/classify_profiles.json`

```json
{
  "profiles": [
    {
      "workflow_name": "classify-docs",
      "name": "default",
      "description": "Default classify workflow profile with standard clarity threshold (0.7)",
      "stages": [
        {
          "stage_name": "init",
          "system_prompt": ""
        },
        {
          "stage_name": "detect",
          "system_prompt": "You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"page_number\": <integer>,\n\t\"markings_found\": [\n\t\t{\n\t\t\t\"text\": \"<exact marking text>\",\n\t\t\t\"location\": \"<header|footer|margin|body>\",\n\t\t\t\"confidence\": <0.0-1.0>,\n\t\t\t\"faded\": <boolean>\n\t\t}\n\t],\n\t\"clarity_score\": <0.0-1.0>,\n\t\"filter_suggestion\": {\n\t\t\"brightness\": <optional integer 0-200>,\n\t\t\"contrast\": <optional integer -100 to 100>,\n\t\t\"saturation\": <optional integer 0-200>\n\t} or null\n}\n\nINSTRUCTIONS:\n- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)\n- Note the location of each marking (header, footer, margin, or body)\n- Assess confidence based on readability (1.0 = perfectly clear, 0.0 = illegible)\n- Set faded=true if the marking appears washed out or hard to read\n- clarity_score reflects overall page quality for marking detection\n- If clarity_score < 0.7, suggest filter adjustment that might improve readability\n- JSON response only; no preamble or dialog"
        },
        {
          "stage_name": "enhance",
          "system_prompt": "You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"page_number\": <integer>,\n\t\"markings_found\": [\n\t\t{\n\t\t\t\"text\": \"<exact marking text>\",\n\t\t\t\"location\": \"<header|footer|margin|body>\",\n\t\t\t\"confidence\": <0.0-1.0>,\n\t\t\t\"faded\": <boolean>\n\t\t}\n\t],\n\t\"clarity_score\": <0.0-1.0>,\n\t\"filter_suggestion\": {\n\t\t\"brightness\": <optional integer 0-200>,\n\t\t\"contrast\": <optional integer -100 to 100>,\n\t\t\"saturation\": <optional integer 0-200>\n\t} or null\n}\n\nINSTRUCTIONS:\n- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)\n- Note the location of each marking (header, footer, margin, or body)\n- Assess confidence based on readability (1.0 = perfectly clear, 0.0 = illegible)\n- Set faded=true if the marking appears washed out or hard to read\n- clarity_score reflects overall page quality for marking detection\n- If clarity_score < 0.7, suggest filter adjustment that might improve readability\n- JSON response only; no preamble or dialog",
          "options": {"clarity_threshold": 0.7}
        },
        {
          "stage_name": "classify",
          "system_prompt": "You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"classification\": \"<overall classification level>\",\n\t\"alternative_readings\": [\n\t\t{\n\t\t\t\"classification\": \"<alternative classification>\",\n\t\t\t\"probability\": <0.0-1.0>,\n\t\t\t\"reason\": \"<why this could be the correct classification>\"\n\t\t}\n\t],\n\t\"marking_summary\": [\"<list of unique markings found>\"],\n\t\"rationale\": \"<explanation of classification decision>\"\n}\n\nINSTRUCTIONS:\n- Analyze all marking detections provided\n- Determine the HIGHEST classification level present\n- If markings are inconsistent or ambiguous, list alternative readings\n- marking_summary should list unique marking texts (deduplicated)\n- rationale should explain how you determined the classification\n- Consider marking confidence and consistency across pages\n- JSON response only; no preamble or dialog"
        },
        {
          "stage_name": "score",
          "system_prompt": "You are a confidence scoring specialist. Evaluate the quality and reliability of document classification results.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"overall_score\": <0.0-1.0>,\n\t\"factors\": [\n\t\t{\n\t\t\t\"name\": \"<factor name>\",\n\t\t\t\"score\": <0.0-1.0>,\n\t\t\t\"weight\": <weight>,\n\t\t\t\"description\": \"<explanation of score>\"\n\t\t}\n\t],\n\t\"recommendation\": \"<ACCEPT|REVIEW|REJECT>\"\n}\n\nFACTORS TO EVALUATE:\n- marking_clarity (weight: 0.30): Average clarity across pages\n- marking_consistency (weight: 0.25): Marking agreement across pages\n- spatial_coverage (weight: 0.15): Markings in expected locations (header/footer)\n- enhancement_impact (weight: 0.10): Value added by enhancement (if applied)\n- alternative_count (weight: 0.10): Fewer alternatives = higher confidence\n- detection_confidence (weight: 0.10): Average marking confidence\n\nTHRESHOLDS:\n- >= 0.90: ACCEPT - Classification is reliable\n- 0.70-0.89: REVIEW - Human verification recommended\n- < 0.70: REJECT - Insufficient confidence\n\nJSON response only; no preamble or dialog"
        }
      ]
    },
    {
      "workflow_name": "classify-docs",
      "name": "aggressive-enhancement",
      "description": "Lower clarity threshold (0.5) for more aggressive enhancement triggering",
      "stages": [
        {
          "stage_name": "enhance",
          "options": {"clarity_threshold": 0.5}
        }
      ]
    }
  ]
}
```

---

## Validation Steps

### Phase 1 Validation (go-agents-orchestration)

1. Run existing tests:
   ```bash
   cd ~/code/go-agents-orchestration
   go test ./...
   ```

2. Verify secrets excluded from JSON:
   ```go
   s := state.New(nil).SetSecret("token", "secret123")
   data, _ := json.Marshal(s)
   // data should NOT contain "secret123"
   ```

### Phase 2 Validation (agent-lab)

1. Run `go mod tidy` after updating dependency

2. Run `go vet ./...` to check for errors

3. Execute a workflow via API and verify token NOT in:
   - Query `checkpoints` table: `SELECT state_data FROM checkpoints`
   - Query `stages` table: `SELECT input_snapshot, output_snapshot FROM stages`
   - Query `runs` table: `SELECT result FROM runs`
   - Observe SSE events for absence of token

### Phase 3 Validation (cmd/seed)

1. Build and run seed command:
   ```bash
   go run ./cmd/seed -dsn "postgres://..." -profiles
   ```

2. Verify profiles created:
   ```sql
   SELECT * FROM profiles WHERE workflow_name = 'classify-docs';
   SELECT * FROM profile_stages WHERE profile_id IN (SELECT id FROM profiles WHERE workflow_name = 'classify-docs');
   ```

3. Test re-running (idempotent):
   ```bash
   go run ./cmd/seed -dsn "postgres://..." -profiles
   # Should succeed without duplicate key errors
   ```

4. Test external file override:
   ```bash
   go run ./cmd/seed -dsn "postgres://..." -profiles -file ./custom.json
   ```
