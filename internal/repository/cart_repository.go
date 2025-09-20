package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/nikolayk812/sqlcpp-demo/internal/db"
	"github.com/nikolayk812/sqlcpp-demo/internal/domain"
	"github.com/nikolayk812/sqlcpp-demo/internal/port"
	"golang.org/x/text/currency"
)

type cartRepository struct {
	q    *db.Queries
	dbtx db.DBTX
}

func NewCart(dbtx db.DBTX) (port.CartRepository, error) {
	if dbtx == nil {
		return nil, fmt.Errorf("dbtx is nil")
	}

	return &cartRepository{
		q:    db.New(dbtx),
		dbtx: dbtx,
	}, nil
}

func (r *cartRepository) GetCart(ctx context.Context, ownerID string) (domain.Cart, error) {
	var cart domain.Cart

	if ownerID == "" {
		return cart, fmt.Errorf("ownerID is empty")
	}

	rows, err := r.q.GetCart(ctx, ownerID)
	if err != nil {
		return cart, fmt.Errorf("q.GetCart: %w", err)
	}

	items, err := mapGetCartRowsToDomain(rows)
	if err != nil {
		return cart, fmt.Errorf("mapGetCartRowsToDomain: %w", err)
	}

	return domain.Cart{
		OwnerID: ownerID,
		Items:   items,
	}, nil
}

func (r *cartRepository) AddItem(ctx context.Context, ownerID string, item domain.CartItem) error {
	if ownerID == "" {
		return fmt.Errorf("ownerID is empty")
	}

	err := r.q.AddItem(ctx, db.AddItemParams{
		OwnerID:       ownerID,
		ProductID:     item.ProductID,
		PriceAmount:   item.Price.Amount,
		PriceCurrency: item.Price.Currency.String(),
	})
	if err != nil {
		return fmt.Errorf("q.AddItem: %w", err)
	}

	return nil
}

func (r *cartRepository) DeleteItem(ctx context.Context, ownerID string, productID uuid.UUID) (bool, error) {
	if ownerID == "" {
		return false, fmt.Errorf("ownerID is empty")
	}

	if productID == uuid.Nil {
		return false, fmt.Errorf("productID is empty")
	}

	rowsAffected, err := r.q.DeleteItem(ctx, db.DeleteItemParams{
		OwnerID:   ownerID,
		ProductID: productID,
	})
	if err != nil {
		return false, fmt.Errorf("q.DeleteItem: %w", err)
	}

	return rowsAffected > 0, nil
}

func mapGetCartRowsToDomain(rows []db.GetCartRow) ([]domain.CartItem, error) {
	var items []domain.CartItem

	for _, row := range rows {
		item, err := mapGetCartRowToDomain(row)
		if err != nil {
			return nil, fmt.Errorf("mapGetCartRowToDomain: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

func mapGetCartRowToDomain(row db.GetCartRow) (domain.CartItem, error) {
	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("currency[%s] is not valid: %w", row.PriceCurrency, err)
	}

	return domain.CartItem{
		ProductID: row.ProductID,
		Price: domain.Money{
			Amount:   row.PriceAmount,
			Currency: parsedCurrency,
		},
		CreatedAt: row.CreatedAt,
	}, nil
}
