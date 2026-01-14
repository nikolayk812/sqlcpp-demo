package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nikolayk812/sqlcpp-demo/internal/db"
	"github.com/nikolayk812/sqlcpp-demo/internal/domain"
	"github.com/nikolayk812/sqlcpp-demo/internal/port"
	"golang.org/x/text/currency"
)

var (
	ErrNotFound = errors.New("not found")
)

type cartRepository struct {
	q    *db.Queries
	dbtx db.DBTX
}

// NewCart creates a new CartRepository with the given dbtx (pgx.Tx or pgxpool.Pool).
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

	cart.OwnerID = ownerID
	cart.Items = make([]domain.CartItem, 0, len(rows))

	for _, row := range rows {
		item, err := mapGetCartRowToDomainCartItem(row)
		if err != nil {
			return cart, fmt.Errorf("mapGetCartRowToDomainCartItem: %w", err)
		}
		cart.Items = append(cart.Items, item)
	}

	return cart, nil
}

func (r *cartRepository) AddItem(ctx context.Context, ownerID string, item domain.CartItem) error {
	if ownerID == "" {
		return fmt.Errorf("ownerID is empty")
	}

	if item.ProductID == uuid.Nil {
		return fmt.Errorf("item.ProductID is empty")
	}

	params := mapDomainCartItemToAddItemParams(ownerID, item)

	err := r.q.AddItem(ctx, params)
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

// mapGetCartRowToDomainCartItem maps SQLC GetCartRow to domain CartItem
func mapGetCartRowToDomainCartItem(row db.GetCartRow) (domain.CartItem, error) {
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

// mapDomainCartItemToAddItemParams maps domain CartItem to SQLC AddItemParams
func mapDomainCartItemToAddItemParams(ownerID string, item domain.CartItem) db.AddItemParams {
	return db.AddItemParams{
		OwnerID:       ownerID,
		ProductID:     item.ProductID,
		PriceAmount:   item.Price.Amount,
		PriceCurrency: item.Price.Currency.String(),
	}
}
