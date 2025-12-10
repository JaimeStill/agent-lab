package documents

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/JaimeStill/agent-lab/internal/storage"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/google/uuid"
)

type repo struct {
	db         *sql.DB
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a document repository with database and blob storage integration.
func New(db *sql.DB, storage storage.System, logger *slog.Logger, pagination pagination.Config) System {
	return &repo{
		db:         db,
		storage:    storage,
		logger:     logger.With("system", "documents"),
		pagination: pagination,
	}
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Document], error) {
	page.Normalize(r.pagination)

	qb := query.
		NewBuilder(projection, defaultSort).
		WhereSearch(page.Search, "Name", "Filename")

	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	docs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanDocument)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}

	result := pagination.NewPageResult(docs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Document, error) {
	q, args := query.
		NewBuilder(projection).
		BuildSingle("Id", id)

	doc, err := repository.QueryOne(ctx, r.db, q, args, scanDocument)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &doc, nil
}

func (r *repo) Create(ctx context.Context, cmd CreateCommand) (*Document, error) {
	id := uuid.New()
	storageKey := buildStorageKey(id, cmd.Filename)

	if err := r.storage.Store(ctx, storageKey, cmd.Data); err != nil {
		return nil, fmt.Errorf("store file: %w", err)
	}

	q := `INSERT INTO documents(id, name, filename, content_type, size_bytes, page_count, storage_key)
		Values($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, filename, content_type, size_bytes, page_count, storage_key, created_at, updated_at`

	doc, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, []any{
			id, cmd.Name, cmd.Filename, cmd.ContentType, cmd.SizeBytes, cmd.PageCount, storageKey,
		}, scanDocument)
	})

	if err != nil {
		if delErr := r.storage.Delete(ctx, storageKey); delErr != nil {
			r.logger.Error("cleanup failed after db error", "storage_key", storageKey, "error", delErr)
		}
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document created", "id", doc.ID, "name", doc.Name, "storage_key", storageKey)
	return &doc, nil
}

func (r *repo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCommand) (*Document, error) {
	q := `UPDATE documents SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, filename, content_type, size_bytes, page_count, storage_key, created_at, updated_at`

	doc, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Document, error) {
		return repository.QueryOne(ctx, tx, q, []any{cmd.Name, id}, scanDocument)
	})

	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	r.logger.Info("document updated", "id", doc.ID, "name", doc.Name)
	return &doc, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	doc, err := r.Find(ctx, id)
	if err != nil {
		if err == ErrNotFound {
			return nil
		}
		return err
	}

	q := `DELETE FROM documents WHERE id = $1`
	_, err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		return struct{}{}, repository.ExecExpectOne(ctx, tx, q, id)
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if err := r.storage.Delete(ctx, doc.StorageKey); err != nil {
		r.logger.Error("storage cleanup failed", "storage_key", doc.StorageKey, "error", err)
	}

	r.logger.Info("document deleted", "id", id)
	return nil
}

func buildStorageKey(id uuid.UUID, filename string) string {
	return fmt.Sprintf("documents/%s/%s", id.String(), sanitizeFilename(filename))
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
