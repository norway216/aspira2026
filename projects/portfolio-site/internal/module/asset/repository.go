package asset

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles database operations for assets.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new asset repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindAll returns all assets.
func (r *Repository) FindAll(ctx context.Context, usageType string, page, pageSize int) ([]Asset, int64, error) {
	where := ""
	args := []interface{}{}
	if usageType != "" {
		where = "WHERE usage_type = ?"
		args = append(args, usageType)
	}

	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM assets %s", where)
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		"SELECT id, filename, original_name, path, mime_type, size_bytes, usage_type, alt_text, created_at FROM assets %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		where,
	)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		var origName, mime, usage, alt sql.NullString
		if err := rows.Scan(&a.ID, &a.Filename, &origName, &a.Path, &mime, &a.SizeBytes, &usage, &alt, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		a.OriginalName = origName.String
		a.MimeType = mime.String
		a.UsageType = usage.String
		a.AltText = alt.String
		assets = append(assets, a)
	}
	return assets, total, rows.Err()
}

// FindByID finds an asset by ID.
func (r *Repository) FindByID(ctx context.Context, id int64) (*Asset, error) {
	query := `SELECT id, filename, original_name, path, mime_type, size_bytes, usage_type, alt_text, created_at
		FROM assets WHERE id = ?`

	var a Asset
	var origName, mime, usage, alt sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.Filename, &origName, &a.Path, &mime, &a.SizeBytes, &usage, &alt, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("asset not found")
	}
	if err != nil {
		return nil, err
	}
	a.OriginalName = origName.String
	a.MimeType = mime.String
	a.UsageType = usage.String
	a.AltText = alt.String
	return &a, nil
}

// Create inserts a new asset record.
func (r *Repository) Create(ctx context.Context, a *Asset) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO assets (filename, original_name, path, mime_type, size_bytes, usage_type, alt_text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		a.Filename, a.OriginalName, a.Path, a.MimeType, a.SizeBytes, a.UsageType, a.AltText,
	)
	if err != nil {
		return fmt.Errorf("create asset: %w", err)
	}
	id, _ := result.LastInsertId()
	a.ID = id
	return nil
}

// Delete removes an asset record by ID.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM assets WHERE id = ?", id)
	return err
}
