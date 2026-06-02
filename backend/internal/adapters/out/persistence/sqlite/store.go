package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"pokemon-binder-finder/internal/domain"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		PRAGMA journal_mode=WAL;
		CREATE TABLE IF NOT EXISTS cards (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, set_id TEXT NOT NULL,
			set_name TEXT NOT NULL, number TEXT NOT NULL, image_url TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS search_jobs (
			id TEXT PRIMARY KEY, card_id TEXT NOT NULL, listing_query TEXT NOT NULL,
			status TEXT NOT NULL, listings_found INTEGER NOT NULL, images_analyzed INTEGER NOT NULL,
			images_total INTEGER NOT NULL, confirmed_matches INTEGER NOT NULL,
			possible_matches INTEGER NOT NULL, vision_degraded INTEGER NOT NULL,
			error TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT
		);
		CREATE TABLE IF NOT EXISTS search_results (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL, listing_id TEXT NOT NULL,
			listing_url TEXT NOT NULL, listing_title TEXT NOT NULL, source_image_url TEXT NOT NULL,
			cached_image_url TEXT NOT NULL, crop_image_url TEXT NOT NULL, polygon_json TEXT NOT NULL,
			confidence REAL NOT NULL, bucket TEXT NOT NULL, reason TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS search_results_job_bucket ON search_results(job_id, bucket, confidence DESC);
	`)
	return err
}

func (s *Store) UpsertCards(ctx context.Context, cards []domain.Card) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO cards(id,name,set_id,set_name,number,image_url)
		VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,set_id=excluded.set_id,
		set_name=excluded.set_name,number=excluded.number,image_url=excluded.image_url`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, card := range cards {
		if _, err := stmt.ExecContext(ctx, card.ID, card.Name, card.SetID, card.SetName, card.Number, card.ImageURL); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SearchCards(ctx context.Context, query string, limit int) ([]domain.Card, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,set_id,set_name,number,image_url FROM cards
		WHERE name LIKE ? OR set_name LIKE ? OR number LIKE ? ORDER BY name,set_name,number LIMIT ?`,
		"%"+query+"%", "%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []domain.Card
	for rows.Next() {
		var card domain.Card
		if err := rows.Scan(&card.ID, &card.Name, &card.SetID, &card.SetName, &card.Number, &card.ImageURL); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func (s *Store) GetCard(ctx context.Context, id string) (domain.Card, error) {
	var card domain.Card
	err := s.db.QueryRowContext(ctx, `SELECT id,name,set_id,set_name,number,image_url FROM cards WHERE id=?`, id).
		Scan(&card.ID, &card.Name, &card.SetID, &card.SetName, &card.Number, &card.ImageURL)
	return card, err
}

func (s *Store) CreateJob(ctx context.Context, job domain.SearchJob) error {
	return s.writeJob(ctx, `INSERT INTO search_jobs VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, job)
}

func (s *Store) UpdateJob(ctx context.Context, job domain.SearchJob) error {
	return s.writeJob(ctx, `UPDATE search_jobs SET card_id=?,listing_query=?,status=?,listings_found=?,
		images_analyzed=?,images_total=?,confirmed_matches=?,possible_matches=?,vision_degraded=?,error=?,
		created_at=?,completed_at=? WHERE id=?`, job, true)
}

func (s *Store) writeJob(ctx context.Context, query string, job domain.SearchJob, update ...bool) error {
	var completed any
	if job.CompletedAt != nil {
		completed = job.CompletedAt.Format(time.RFC3339Nano)
	}
	args := []any{job.ID, job.CardID, job.ListingQuery, job.Status, job.ListingsFound, job.ImagesAnalyzed,
		job.ImagesTotal, job.ConfirmedMatches, job.PossibleMatches, job.VisionDegraded, job.Error,
		job.CreatedAt.Format(time.RFC3339Nano), completed}
	if len(update) > 0 {
		args = append(args[1:], job.ID)
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) GetJob(ctx context.Context, id string) (domain.SearchJob, error) {
	var job domain.SearchJob
	var created string
	var completed sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id,card_id,listing_query,status,listings_found,images_analyzed,
		images_total,confirmed_matches,possible_matches,vision_degraded,error,created_at,completed_at
		FROM search_jobs WHERE id=?`, id).Scan(&job.ID, &job.CardID, &job.ListingQuery, &job.Status,
		&job.ListingsFound, &job.ImagesAnalyzed, &job.ImagesTotal, &job.ConfirmedMatches,
		&job.PossibleMatches, &job.VisionDegraded, &job.Error, &created, &completed)
	if err != nil {
		return job, err
	}
	job.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if completed.Valid {
		value, _ := time.Parse(time.RFC3339Nano, completed.String)
		job.CompletedAt = &value
	}
	return job, nil
}

func (s *Store) SaveResult(ctx context.Context, result domain.SearchResult) error {
	polygon, _ := json.Marshal(result.Polygon)
	_, err := s.db.ExecContext(ctx, `INSERT INTO search_results VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET confidence=excluded.confidence,bucket=excluded.bucket,reason=excluded.reason`,
		result.ID, result.JobID, result.ListingID, result.ListingURL, result.ListingTitle, result.SourceImageURL,
		result.CachedImageURL, result.CropImageURL, string(polygon), result.Confidence, result.Bucket, result.Reason)
	return err
}

func (s *Store) ListResults(ctx context.Context, jobID, bucket string) ([]domain.SearchResult, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,job_id,listing_id,listing_url,listing_title,source_image_url,
		cached_image_url,crop_image_url,polygon_json,confidence,bucket,reason FROM search_results
		WHERE job_id=? AND bucket=? ORDER BY confidence DESC`, jobID, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []domain.SearchResult
	for rows.Next() {
		var result domain.SearchResult
		var polygon string
		if err := rows.Scan(&result.ID, &result.JobID, &result.ListingID, &result.ListingURL, &result.ListingTitle,
			&result.SourceImageURL, &result.CachedImageURL, &result.CropImageURL, &polygon,
			&result.Confidence, &result.Bucket, &result.Reason); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(polygon), &result.Polygon)
		results = append(results, result)
	}
	return results, rows.Err()
}

func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite: %w", err)
	}
	return nil
}
