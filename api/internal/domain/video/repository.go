package video

import "context"

type Repository interface {
	Create(ctx context.Context, video *Video) error
	GetByID(ctx context.Context, id uint) (*Video, error)
	GetAll(ctx context.Context) ([]*Video, error)
	Update(ctx context.Context, video *Video) error
	Delete(ctx context.Context, id uint) error
}
