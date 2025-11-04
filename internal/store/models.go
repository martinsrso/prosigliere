package store

import (
	"sync"
	"time"
)

type BlogPost struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Comments  []Comment `json:"comments,omitempty"`
}

type Comment struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu          sync.RWMutex
	posts       map[int]*BlogPost
	comments    map[int][]Comment
	nextPostID  int
	nextCommentID int
}

func NewStore() *Store {
	return &Store{
		posts:        make(map[int]*BlogPost),
		comments:     make(map[int][]Comment),
		nextPostID:   1,
		nextCommentID: 1,
	}
}

func (s *Store) GetAllPosts() ([]BlogPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	posts := make([]BlogPost, 0, len(s.posts))
	for _, post := range s.posts {
		postCopy := *post
		postCopy.Comments = nil
		posts = append(posts, postCopy)
	}
	return posts, nil
}

func (s *Store) GetCommentCount(postID int) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.comments[postID]), nil
}

func (s *Store) GetPost(id int) (*BlogPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	post, exists := s.posts[id]
	if !exists {
		return nil, ErrPostNotFound
	}

	postCopy := *post
	postCopy.Comments = s.comments[id]
	return &postCopy, nil
}

func (s *Store) CreatePost(title, content string) (*BlogPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextPostID
	s.nextPostID++

	post := &BlogPost{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	s.posts[id] = post
	s.comments[id] = make([]Comment, 0)
	return post, nil
}

func (s *Store) AddComment(postID int, content string) (*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.posts[postID]; !exists {
		return nil, ErrPostNotFound
	}

	id := s.nextCommentID
	s.nextCommentID++

	comment := Comment{
		ID:        id,
		PostID:    postID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	s.comments[postID] = append(s.comments[postID], comment)
	return &comment, nil
}
