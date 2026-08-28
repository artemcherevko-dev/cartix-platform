package server

import (
	"context"
	"log"

	"catalog/internal/db"

	catalogpb "proto/catalog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Handler struct {
	catalogpb.UnimplementedCatalogServer
	repo *db.Repo
}

func NewHandler(repo *db.Repo) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetProduct(ctx context.Context, req *catalogpb.GetProductReq) (*catalogpb.ProductRes, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	product, err := h.repo.GetByID(ctx, id)
	if err != nil {
		return nil, toStatus(err)
	}

	return productToProto(product), nil
}

func (h *Handler) GetProducts(ctx context.Context, req *catalogpb.GetProductsReq) (*catalogpb.GetProductsRes, error) {
	ids := make([]uuid.UUID, 0, len(req.Ids))
	for _, raw := range req.Ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid product id: "+raw)
		}
		ids = append(ids, id)
	}

	products, err := h.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "get products failed")
	}

	res := &catalogpb.GetProductsRes{Products: make([]*catalogpb.ProductRes, 0, len(products))}
	for _, product := range products {
		res.Products = append(res.Products, productToProto(&product))
	}

	return res, nil
}

func (h *Handler) ListProducts(ctx context.Context, req *catalogpb.ListProductsReq) (*catalogpb.ListProductsRes, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}

	products, total, err := h.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "list products failed")
	}

	res := &catalogpb.ListProductsRes{
		Products: make([]*catalogpb.ProductRes, 0, len(products)),
		Total:    int32(total),
	}
	for _, product := range products {
		res.Products = append(res.Products, productToProto(&product))
	}

	return res, nil
}

func productToProto(p *db.Product) *catalogpb.ProductRes {
	var attrs *structpb.Struct
	if p.Attributes != nil {
		attrs, _ = structpb.NewStruct(p.Attributes)
	}

	return &catalogpb.ProductRes{
		Id:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		PriceMinor:  p.PriceMinor,
		Currency:    p.Currency,
		Category:    p.Category,
		Attributes:  attrs,
		Active:      p.Active,
	}
}

func (h *Handler) CreateProduct(ctx context.Context, req *catalogpb.CreateProductReq) (*catalogpb.ProductRes, error) {
	if req.Name == "" || req.PriceMinor <= 0 || req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "name, price_minor and currency are required")
	}

	var attrs db.Attributes
	if req.Attributes != nil {
		attrs = db.Attributes(req.Attributes.AsMap())
	}

	product, err := h.repo.Create(ctx, db.Product{
		Name:        req.Name,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
		Category:    req.Category,
		Attributes:  attrs,
		Active:      req.Active,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return productToProto(product), nil
}

func (h *Handler) UpdateProduct(ctx context.Context, req *catalogpb.UpdateProductReq) (*catalogpb.ProductRes, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}
	if req.Name == "" || req.PriceMinor <= 0 || req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "name, price_minor and currency are required")
	}

	var attrs db.Attributes
	if req.Attributes != nil {
		attrs = db.Attributes(req.Attributes.AsMap())
	}

	product, err := h.repo.Update(ctx, id, db.UpdateInput{
		Name:        req.Name,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
		Category:    req.Category,
		Attributes:  attrs,
		Active:      req.Active,
	})
	if err != nil {
		return nil, toStatus(err)
	}

	return productToProto(product), nil
}

func (h *Handler) DeleteProduct(ctx context.Context, req *catalogpb.DeleteProductReq) (*catalogpb.DeleteProductRes, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	deleted, err := h.repo.Delete(ctx, id)
	if err != nil {
		return nil, toStatus(err)
	}

	return &catalogpb.DeleteProductRes{Deleted: deleted}, nil
}

func toStatus(err error) error {
	if err == db.ErrProductNotFound {
		return status.Error(codes.NotFound, "product not found")
	}

	log.Printf("[CATALOG] repo error: %v", err)
	return status.Error(codes.Internal, "internal error")
}
