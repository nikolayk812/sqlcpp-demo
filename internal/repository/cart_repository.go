package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp-demo/internal/db"
	"github.com/nikolayk812/sqlcpp-demo/internal/domain"
	"github.com/nikolayk812/sqlcpp-demo/internal/port"
	"golang.org/x/text/currency"
)

type cartRepository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewCart(pool *pgxpool.Pool) port.CartRepository {
	return &cartRepository{
		q:    db.New(pool),
		pool: pool,
	}
}

func NewCartWithTx(tx pgx.Tx) port.CartRepository {
	return &cartRepository{
		q:    db.New(tx),
		pool: nil, // use provided transaction instead
	}
}

func (r *cartRepository) GetCart(ctx context.Context, ownerID string) (domain.Cart, error) {
	if ownerID == "" {
		return domain.Cart{}, fmt.Errorf("ownerID is empty")
	}

	dbCartItems, err := r.q.GetCart(ctx, ownerID)
	if err != nil {
		return domain.Cart{}, fmt.Errorf("q.GetCart: %w", err)
	}

	items, err := mapGetCartRowsToDomain(dbCartItems)
	if err != nil {
		return domain.Cart{}, fmt.Errorf("mapGetCartRowsToDomain: %w", err)
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

	rowsAffected, err := r.q.DeleteItem(ctx, db.DeleteItemParams{
		OwnerID:   ownerID,
		ProductID: productID,
	})
	if err != nil {
		return false, fmt.Errorf("q.DeleteItem: %w", err)
	}

	return rowsAffected > 0, nil
}

func mapGetCartRowToDomain(row db.GetCartRow) (domain.CartItem, error) {
	parsedCurrency, err := currency.ParseISO(row.PriceCurrency)
	if err != nil {
		return domain.CartItem{}, fmt.Errorf("currency[%s] is not valid: %w", row.PriceCurrency, err)
	}

	return domain.CartItem{
		ProductID: row.ProductID,
		Price:     domain.Money{Amount: row.PriceAmount, Currency: parsedCurrency},
		CreatedAt: row.CreatedAt,
	}, nil
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