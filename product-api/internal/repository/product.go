package repository

import (
	"context"
	"github.com/kahvecikaan/buildingMicroservices/product-api/internal/domain"
	"sync"
)

type ProductRepository interface {
	GetAll(ctx context.Context) ([]*domain.Product, error)
	GetById(ctx context.Context, id int) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Add(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id int) error
}

type memoryProductRepository struct {
	products []*domain.Product
	mutex    sync.RWMutex
}

func NewMemoryProductRepository() ProductRepository {
	return &memoryProductRepository{
		products: []*domain.Product{
			{
				ID:          1,
				Name:        "Latte",
				Description: "Frothy milky coffee",
				Price:       2.45,
				SKU:         "abc323",
			},
			{
				ID:          2,
				Name:        "Espresso",
				Description: "Short and strong coffee without milk",
				Price:       1.99,
				SKU:         "fjd34",
			},
		},
	}
}

func (r *memoryProductRepository) GetAll(ctx context.Context) ([]*domain.Product, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.products, nil
}

func (r *memoryProductRepository) GetById(ctx context.Context, id int) (*domain.Product, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, product := range r.products {
		if product.ID == id {
			return product, nil
		}
	}

	return nil, domain.ErrProductNotFound
}

func (r *memoryProductRepository) Update(ctx context.Context, product *domain.Product) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, p := range r.products {
		if p.ID == product.ID {
			r.products[i] = product
			return nil
		}
	}

	return domain.ErrProductNotFound
}

func (r *memoryProductRepository) Add(ctx context.Context, product *domain.Product) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	product.ID = r.getNextID()
	r.products = append(r.products, product)
	return nil
}

func (r *memoryProductRepository) Delete(ctx context.Context, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, product := range r.products {
		if product.ID == id {
			r.products = append(r.products[:i], r.products[i+1:]...)
			return nil
		}
	}

	return domain.ErrProductNotFound
}

func (r *memoryProductRepository) getNextID() int {
	if len(r.products) == 0 {
		return 1
	}
	return r.products[len(r.products)-1].ID + 1
}
