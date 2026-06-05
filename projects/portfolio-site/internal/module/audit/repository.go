package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository handles database operations for audit logs.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new audit repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create records a new audit log entry.
func (r *Repository) Create(ctx context.Context, entry *Log) error {
	query := `INSERT INTO audit_logs (actor_id, actor_name, action, resource_type, resource_id, detail, ip_address, user_agent, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		entry.ActorID, entry.ActorName, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.Detail,
		entry.IPAddress, entry.UserAgent, entry.Status, now,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	id, _ := result.LastInsertId()
	entry.ID = id
	entry.CreatedAt = now
	return nil
}

// Find returns audit logs matching the query.
func (r *Repository) Find(ctx context.Context, q Query) ([]Log, int64, error) {
	where := []string{}
	args := []interface{}{}

	if q.ActorID > 0 {
		where = append(where, "actor_id = ?")
		args = append(args, q.ActorID)
	}
	if q.Action != "" {
		where = append(where, "action = ?")
		args = append(args, q.Action)
	}
	if q.ResourceType != "" {
		where = append(where, "resource_type = ?")
		args = append(args, q.ResourceType)
	}
	if q.ResourceID > 0 {
		where = append(where, "resource_id = ?")
		args = append(args, q.ResourceID)
	}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}
	if q.DateFrom != "" {
		where = append(where, "date(created_at) >= ?")
		args = append(args, q.DateFrom)
	}
	if q.DateTo != "" {
		where = append(where, "date(created_at) <= ?")
		args = append(args, q.DateTo)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count.
	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	// Query.
	orderBy := q.OrderBy
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, actor_id, actor_name, action, resource_type, resource_id,
		detail, ip_address, user_agent, status, created_at
		FROM audit_logs %s ORDER BY %s LIMIT ? OFFSET ?`,
		whereClause, orderBy,
	)
	args = append(args, q.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []Log
	for rows.Next() {
		var l Log
		var actorID, resourceID sql.NullInt64
		var actorName, detail, ipAddr, userAgent sql.NullString

		if err := rows.Scan(
			&l.ID, &actorID, &actorName, &l.Action,
			&l.ResourceType, &resourceID,
			&detail, &ipAddr, &userAgent,
			&l.Status, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}

		if actorID.Valid {
			l.ActorID = actorID.Int64
		}
		l.ActorName = actorName.String
		l.Detail = detail.String
		l.IPAddress = ipAddr.String
		l.UserAgent = userAgent.String
		if resourceID.Valid {
			l.ResourceID = resourceID.Int64
		}

		logs = append(logs, l)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetActionSummary returns counts grouped by action type.
func (r *Repository) GetActionSummary(ctx context.Context, days int) (map[string]int64, error) {
	query := `SELECT action, COUNT(*) as cnt FROM audit_logs
		WHERE created_at >= datetime('now', ?)
		GROUP BY action ORDER BY cnt DESC`

	rows, err := r.db.QueryContext(ctx, query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[string]int64)
	for rows.Next() {
		var action string
		var count int64
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		summary[action] = count
	}
	return summary, rows.Err()
}

// CleanOldRecords removes audit logs older than the specified days.
func (r *Repository) CleanOldRecords(ctx context.Context, olderThanDays int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM audit_logs WHERE created_at < datetime('now', ?)",
		fmt.Sprintf("-%d days", olderThanDays),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
