package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"

	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/pkg/pagination"
	"github.com/JaimeStill/agent-lab/pkg/query"
	"github.com/JaimeStill/agent-lab/pkg/repository"
	"github.com/JaimeStill/agent-lab/pkg/storage"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/image"
	"github.com/google/uuid"
)

type renderTask struct {
	pageNum int
	result  *Image
	err     error
}

type repo struct {
	db         *sql.DB
	documents  documents.System
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
}

// New creates a new image management system.
func New(
	docs documents.System,
	db *sql.DB,
	storage storage.System,
	logger *slog.Logger,
	pagination pagination.Config,
) System {
	return &repo{
		db:         db,
		documents:  docs,
		storage:    storage,
		logger:     logger.With("system", "images"),
		pagination: pagination,
	}
}

func (r *repo) Handler() *Handler {
	return NewHandler(r, r.logger, r.pagination)
}

func (r *repo) List(ctx context.Context, page pagination.PageRequest, filters Filters) (*pagination.PageResult[Image], error) {
	page.Normalize(r.pagination)

	qb := query.NewBuilder(projection, defaultSort)
	filters.Apply(qb)

	if len(page.Sort) > 0 {
		qb.OrderByFields(page.Sort)
	}

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count images: %w", err)
	}

	pageSQL, pageArgs := qb.BuildPage(page.Page, page.PageSize)
	imgs, err := repository.QueryMany(ctx, r.db, pageSQL, pageArgs, scanImage)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}

	result := pagination.NewPageResult(imgs, total, page.Page, page.PageSize)
	return &result, nil
}

func (r *repo) Find(ctx context.Context, id uuid.UUID) (*Image, error) {
	q, args := query.NewBuilder(projection).BuildSingle("ID", id)
	img, err := repository.QueryOne(ctx, r.db, q, args, scanImage)
	if err != nil {
		return nil, repository.MapError(err, ErrNotFound, ErrDuplicate)
	}
	return &img, nil
}

func (r *repo) Data(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	img, err := r.Find(ctx, id)
	if err != nil {
		return nil, "", err
	}

	data, err := r.storage.Retrieve(ctx, img.StorageKey)
	if err != nil {
		return nil, "", fmt.Errorf("retrieve image: %w", err)
	}

	contentType, err := img.Format.MimeType()
	if err != nil {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (r *repo) Render(ctx context.Context, documentID uuid.UUID, opts RenderOptions) ([]Image, error) {
	doc, err := r.documents.Find(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !document.IsSupported(doc.ContentType) {
		return nil, ErrUnsupportedFormat
	}

	if doc.PageCount == nil || *doc.PageCount < 1 {
		return nil, fmt.Errorf("%w: document has no pages to render", ErrRenderFailed)
	}

	pageExpr := opts.Pages
	if pageExpr == "" {
		pageExpr = fmt.Sprintf("1-%d", *doc.PageCount)
	}

	pages, err := ParsePageRange(pageExpr, *doc.PageCount)
	if err != nil {
		return nil, err
	}

	docPath, err := r.storage.Path(ctx, doc.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	workerCount := renderWorkerCount(len(pages))
	tasks := make(chan int, len(pages))
	results := make(chan renderTask, len(pages))

	var wg sync.WaitGroup
	for range workerCount {
		wg.Go(func() {
			r.renderWorker(ctx, documentID, docPath, doc.ContentType, opts, tasks, results)
		})
	}

	for _, pageNum := range pages {
		tasks <- pageNum
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	resultMap := make(map[int]*Image)
	for task := range results {
		if task.err != nil {
			return nil, task.err
		}
		resultMap[task.pageNum] = task.result
	}

	images := make([]Image, 0, len(pages))
	for _, pageNum := range pages {
		if img, ok := resultMap[pageNum]; ok {
			images = append(images, *img)
		}
	}

	return images, nil
}

func (r *repo) Delete(ctx context.Context, id uuid.UUID) error {
	img, err := r.Find(ctx, id)
	if err != nil {
		return err
	}

	q := `DELETE FROM images WHERE id = $1`
	_, err = repository.WithTx(ctx, r.db, func(tx *sql.Tx) (struct{}, error) {
		return struct{}{}, repository.ExecExpectOne(ctx, tx, q, id)
	})

	if err != nil {
		return repository.MapError(err, ErrNotFound, ErrDuplicate)
	}

	if err := r.storage.Delete(ctx, img.StorageKey); err != nil {
		r.logger.Warn("failed to delete image file", "key", img.StorageKey, "error", err)
	}

	return nil
}

func (r *repo) renderPage(ctx context.Context, documentID uuid.UUID, doc document.Document, renderer image.Renderer, pageNum int, opts RenderOptions) (*Image, error) {
	existing, err := r.findExisting(ctx, documentID, pageNum, opts)
	if err != nil {
		return nil, err
	}

	if existing != nil && !opts.Force {
		return existing, nil
	}

	page, err := doc.ExtractPage(pageNum)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	data, err := page.ToImage(renderer, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	storageKey := fmt.Sprintf("images/%s/%s.%s", documentID, uuid.New(), opts.Format)

	if err := r.storage.Store(ctx, storageKey, data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	if existing != nil {
		if err := r.update(ctx, existing.ID, storageKey, int64(len(data))); err != nil {
			r.storage.Delete(ctx, storageKey)
			return nil, err
		}
		return r.Find(ctx, existing.ID)
	}

	img := opts.ToImage(uuid.New(), documentID, pageNum, storageKey, int64(len(data)))

	if err := r.create(ctx, img); err != nil {
		r.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return r.Find(ctx, img.ID)
}

func (r *repo) renderWorker(
	ctx context.Context,
	documentID uuid.UUID,
	docPath string,
	contentType string,
	opts RenderOptions,
	tasks <-chan int,
	results chan<- renderTask,
) {
	openDoc, err := document.Open(docPath, contentType)
	if err != nil {
		for pageNum := range tasks {
			results <- renderTask{
				pageNum: pageNum,
				err:     fmt.Errorf("%w: %v", ErrRenderFailed, err),
			}
		}
		return
	}
	defer openDoc.Close()

	renderer, err := image.NewImageMagickRenderer(opts.ToImageConfig())
	if err != nil {
		for pageNum := range tasks {
			results <- renderTask{
				pageNum: pageNum,
				err:     fmt.Errorf("%w: %v", ErrRenderFailed, err),
			}
		}
		return
	}

	for pageNum := range tasks {
		select {
		case <-ctx.Done():
			results <- renderTask{pageNum: pageNum, err: ctx.Err()}
			return
		default:
		}

		img, err := r.renderPage(ctx, documentID, openDoc, renderer, pageNum, opts)
		results <- renderTask{pageNum: pageNum, result: img, err: err}
	}
}

func (r *repo) findExisting(ctx context.Context, documentID uuid.UUID, pageNum int, opts RenderOptions) (*Image, error) {
	q, args := query.NewBuilder(projection).
		WhereEquals("DocumentID", documentID).
		WhereEquals("PageNumber", pageNum).
		WhereEquals("Format", opts.Format).
		WhereEquals("DPI", opts.DPI).
		WhereNullable("Quality", opts.Quality).
		WhereNullable("Brightness", opts.Brightness).
		WhereNullable("Contrast", opts.Contrast).
		WhereNullable("Saturation", opts.Saturation).
		WhereNullable("Rotation", opts.Rotation).
		WhereNullable("Background", opts.Background).
		BuildSingleOrNull()

	img, err := repository.QueryOne(ctx, r.db, q, args, scanImage)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *repo) create(ctx context.Context, img *Image) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO images (id, document_id, page_number, format, dpi, quality,
			brightness, contrast, saturation, rotation, background, storage_key, size_bytes)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		img.ID, img.DocumentID, img.PageNumber, img.Format, img.DPI, img.Quality,
		img.Brightness, img.Contrast, img.Saturation, img.Rotation, img.Background,
		img.StorageKey, img.SizeBytes,
	)
	return err
}

func (r *repo) update(ctx context.Context, id uuid.UUID, storageKey string, sizeBytes int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE images SET storage_key = $1, size_bytes = $2 WHERE id = $3`,
		storageKey, sizeBytes, id,
	)
	return err
}

func renderWorkerCount(pageCount int) int {
	workers := max(min(runtime.NumCPU(), pageCount), 1)
	return workers
}
