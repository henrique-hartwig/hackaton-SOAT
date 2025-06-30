package video

import "context"

type Repository interface {
	Create(ctx context.Context, video *Video) error
	GetByID(ctx context.Context, id uint) (*Video, error)
	GetAll(ctx context.Context) ([]*Video, error)
	GetByUserID(ctx context.Context, userID uint) ([]*Video, error)
	Update(ctx context.Context, video *Video) error
	Delete(ctx context.Context, id uint) error
}
