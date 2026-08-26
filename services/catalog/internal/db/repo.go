package db

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrProductNotFound = errors.New("product not found")

type Repo struct {
	gorm *gorm.DB
}

func NewRepo(gormDB *gorm.DB) *Repo {
	return &Repo{gorm: gormDB}
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	var product Product
	if err := r.gorm.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	return &product, nil
}

func (r *Repo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]Product, error) {
	var products []Product
	if err := r.gorm.WithContext(ctx).Find(&products, "id IN ?", ids).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func (r *Repo) List(ctx context.Context, page, pageSize int) ([]Product, int64, error) {
	var (
		products []Product
		total    int64
	)

	if err := r.gorm.WithContext(ctx).Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := r.gorm.WithContext(ctx).Order("created_at ASC")
	if pageSize > 0 {
		query = query.Limit(pageSize).Offset(offset)
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *Repo) Create(ctx context.Context, product Product) (*Product, error) {
	product.ID = uuid.Must(uuid.NewV7())

	if err := r.gorm.WithContext(ctx).Create(&product).Error; err != nil {
		return nil, err
	}

	return &product, nil
}

// Update saves mutable fields of an existing product. Returns false when the
// product does not exist.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*Product, error) {
	if _, err := r.GetByID(ctx, id); err != nil {
		return nil, err
	}

	if err := r.gorm.WithContext(ctx).Model(&Product{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"name":        in.Name,
			"description": in.Description,
			"price_minor": in.PriceMinor,
			"currency":    in.Currency,
			"category":    in.Category,
			"attributes":  in.Attributes,
			"active":      in.Active,
		}).Error; err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Delete removes a product. Returns false when the product does not exist.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	result := r.gorm.WithContext(ctx).Delete(&Product{}, "id = ?", id)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}
