package repository_test

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp-demo/internal/domain"
	"github.com/nikolayk812/sqlcpp-demo/internal/port"
	"github.com/nikolayk812/sqlcpp-demo/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/text/currency"
)

type cartRepositorySuite struct {
	suite.Suite

	repo port.CartRepository
	pool *pgxpool.Pool
}

// entry point to run the tests in the suite
func TestCartRepositorySuite(t *testing.T) {
	suite.Run(t, new(cartRepositorySuite))
}

// before all tests in the suite
func (suite *cartRepositorySuite) SetupSuite() {
	ctx := suite.T().Context()

	_, connStr, err := startPostgres(ctx)
	suite.NoError(err)

	suite.pool, err = pgxpool.New(ctx, connStr)
	suite.NoError(err)

	suite.repo, err = repository.NewCart(suite.pool)
	suite.NoError(err)
}

// after all tests in the suite
func (suite *cartRepositorySuite) TearDownSuite() {
	if suite.pool != nil {
		suite.pool.Close()
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		setupFunc func() string
		wantItems int
	}{
		{
			name: "empty cart: ok",
			setupFunc: func() string {
				return gofakeit.UUID()
			},
			wantItems: 0,
		},
		{
			name: "cart with single item: ok",
			setupFunc: func() string {
				ownerID := gofakeit.UUID()
				item := randomCartItem()
				err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
				suite.NoError(err)
				return ownerID
			},
			wantItems: 1,
		},
		{
			name: "cart with multiple items: ok",
			setupFunc: func() string {
				ownerID := gofakeit.UUID()
				for i := range 3 {
					_ = i
					item := randomCartItem()
					err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
					suite.NoError(err)
				}
				return ownerID
			},
			wantItems: 3,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			ownerID := tt.setupFunc()

			cart, err := suite.repo.GetCart(ctx, ownerID)
			require.NoError(t, err)

			require.Equal(t, ownerID, cart.OwnerID)
			require.Len(t, cart.Items, tt.wantItems)

			// Verify each item has required fields
			for _, item := range cart.Items {
				require.NotEqual(t, uuid.Nil, item.ProductID)
				require.True(t, item.Price.Amount.GreaterThan(decimal.Zero))
				require.NotEmpty(t, item.Price.Currency.String())
				require.False(t, item.CreatedAt.IsZero())
			}
		})
	}
}

func (suite *cartRepositorySuite) TestAddItem() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		item      domain.CartItem
		wantError string
	}{
		{
			name:    "add new item: ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
		{
			name:    "add item with same product (upsert): ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// For upsert test, add the item first
			if tt.name == "add item with same product (upsert): ok" {
				err := suite.repo.AddItem(ctx, tt.ownerID, tt.item)
				require.NoError(t, err)

				// Update the price for upsert test
				tt.item.Price.Amount = decimal.NewFromFloat(999.99)
			}

			err := suite.repo.AddItem(ctx, tt.ownerID, tt.item)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the item was added
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			require.Len(t, cart.Items, 1)
			item := cart.Items[0]
			require.Equal(t, tt.item.ProductID, item.ProductID)
			require.True(t, tt.item.Price.Amount.Equal(item.Price.Amount))
			require.Equal(t, tt.item.Price.Currency, item.Price.Currency)
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name         string
		setupFunc    func() (string, uuid.UUID)
		targetIDFunc func(string, uuid.UUID) (string, uuid.UUID) // modify owner/product IDs for test
		wantDeleted  bool
		wantError    string
	}{
		{
			name: "delete existing item: ok",
			setupFunc: func() (string, uuid.UUID) {
				ownerID := gofakeit.UUID()
				item := randomCartItem()
				err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
				suite.NoError(err)
				return ownerID, item.ProductID
			},
			wantDeleted: true,
		},
		{
			name: "delete non-existing item: ok but not deleted",
			setupFunc: func() (string, uuid.UUID) {
				return gofakeit.UUID(), uuid.MustParse(gofakeit.UUID())
			},
			wantDeleted: false,
		},
		{
			name: "delete item with wrong owner: ok but not deleted",
			setupFunc: func() (string, uuid.UUID) {
				ownerID := gofakeit.UUID()
				item := randomCartItem()
				err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
				suite.NoError(err)
				return ownerID, item.ProductID
			},
			targetIDFunc: func(ownerID string, productID uuid.UUID) (string, uuid.UUID) {
				return gofakeit.UUID(), productID // wrong owner
			},
			wantDeleted: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			ownerID, productID := tt.setupFunc()

			targetOwnerID, targetProductID := ownerID, productID
			if tt.targetIDFunc != nil {
				targetOwnerID, targetProductID = tt.targetIDFunc(ownerID, productID)
			}

			deleted, err := suite.repo.DeleteItem(ctx, targetOwnerID, targetProductID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantDeleted, deleted)

			// Verify the item was deleted (or not)
			cart, err := suite.repo.GetCart(ctx, ownerID)
			require.NoError(t, err)

			if tt.wantDeleted {
				require.Empty(t, cart.Items)
			} else if tt.targetIDFunc == nil {
				// If we tried to delete non-existing, cart should still be empty
				require.Empty(t, cart.Items)
			} else {
				// If we tried to delete with wrong owner, item should still exist
				require.Len(t, cart.Items, 1)
			}
		})
	}
}

func (suite *cartRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE cart_items CASCADE")
	suite.NoError(err)
}

func randomCartItem() domain.CartItem {
	price := gofakeit.Price(1, 1000)
	currencyUnit := randomCurrency()

	return domain.CartItem{
		ProductID: uuid.MustParse(gofakeit.UUID()),
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
		CreatedAt: time.Now(), // This will be overridden by database
	}
}

func randomCurrency() currency.Unit {
	var (
		result currency.Unit
		err    error
	)

	for {
		// tag is not a recognized currency
		result, err = currency.ParseISO(gofakeit.CurrencyShort())
		if err == nil {
			break
		}
	}

	return result
}
