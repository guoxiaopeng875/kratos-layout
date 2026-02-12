package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-kratos/kratos-layout/internal/biz"
)

type greeterRepo struct {
	data *Data
	log  *log.Helper
}

// NewGreeterRepo creates a new greeter repository.
func NewGreeterRepo(data *Data, logger log.Logger) biz.GreeterRepo {
	return &greeterRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	// TODO: Implement database save logic
	// Example:
	// model := &GreeterModel{Name: g.Hello}
	// if err := r.data.DB().WithContext(ctx).Create(model).Error; err != nil {
	//     return nil, err
	// }
	// return &biz.Greeter{Hello: model.Name}, nil
	return g, nil
}

func (r *greeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	// TODO: Implement database update logic
	return g, nil
}

func (r *greeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	// TODO: Implement database find logic
	// Example:
	// var model GreeterModel
	// if err := r.data.DB().WithContext(ctx).First(&model, id).Error; err != nil {
	//     if errors.Is(err, gorm.ErrRecordNotFound) {
	//         return nil, biz.ErrGreeterNotFound
	//     }
	//     return nil, err
	// }
	// return &biz.Greeter{Hello: model.Name}, nil
	return nil, nil
}

func (r *greeterRepo) ListByHello(ctx context.Context, hello string) ([]*biz.Greeter, error) {
	// TODO: Implement database list by hello logic
	return nil, nil
}

func (r *greeterRepo) ListAll(ctx context.Context) ([]*biz.Greeter, error) {
	// TODO: Implement database list logic
	return []*biz.Greeter{}, nil
}
