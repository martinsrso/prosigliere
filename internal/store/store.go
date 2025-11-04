package store

type StoreInterface interface {
	GetAllPosts() ([]BlogPost, error)
	GetPost(id int) (*BlogPost, error)
	CreatePost(title, content string) (*BlogPost, error)
	AddComment(postID int, content string) (*Comment, error)
	GetCommentCount(postID int) (int, error)
}
