package database

import (
	"context"
	"database/sql"
	"log/slog"
	"prosig/internal/store"
	"time"

	_ "github.com/lib/pq"
)

type DBStore struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewDBStore(dsn string, logger *slog.Logger) (*DBStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &DBStore{db: db, logger: logger}, nil
}

func (s *DBStore) Close() error {
	return s.db.Close()
}

func (s *DBStore) GetAllPosts() ([]store.BlogPost, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.title, p.content, p.created_at
		FROM posts p
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []store.BlogPost
	for rows.Next() {
		var post store.BlogPost
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *DBStore) GetCommentCount(postID int) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = $1", postID).Scan(&count)
	return count, err
}

func (s *DBStore) GetPost(id int) (*store.BlogPost, error) {
	var post store.BlogPost
	err := s.db.QueryRow(`
		SELECT id, title, content, created_at
		FROM posts
		WHERE id = $1
	`, id).Scan(&post.ID, &post.Title, &post.Content, &post.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, store.ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT id, post_id, content, created_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []store.Comment
	for rows.Next() {
		var comment store.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.Content, &comment.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	post.Comments = comments

	return &post, rows.Err()
}

func (s *DBStore) CreatePost(title, content string) (*store.BlogPost, error) {
	var post store.BlogPost
	err := s.db.QueryRow(`
		INSERT INTO posts (title, content, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, title, content, created_at
	`, title, content, time.Now()).Scan(&post.ID, &post.Title, &post.Content, &post.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *DBStore) AddComment(postID int, content string) (*store.Comment, error) {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)", postID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, store.ErrPostNotFound
	}

	var comment store.Comment
	err = s.db.QueryRow(`
		INSERT INTO comments (post_id, content, created_at)
		VALUES ($1, $2, $3)
		RETURNING id, post_id, content, created_at
	`, postID, content, time.Now()).Scan(&comment.ID, &comment.PostID, &comment.Content, &comment.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &comment, nil
}
