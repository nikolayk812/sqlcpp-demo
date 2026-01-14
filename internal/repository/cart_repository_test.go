package repository_test

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp-demo/internal/domain"
	"github.com/nikolayk812/sqlcpp-demo/internal/port"
	"github.com/nikolayk812/sqlcpp-demo/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
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

func (suite *cartRepositorySuite) TestAddItem() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		item      domain.CartItem
		wantError string
	}{
		{
			name:    "add valid item: ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
		{
			name:    "add item with empty owner ID: ok", // should still work with SQL
			ownerID: "",
			item:    randomCartItem(),
		},
		{
			name:    "add duplicate item (upsert): ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			err := suite.repo.AddItem(ctx, tt.ownerID, tt.item)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the item was added
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			require.Equal(t, tt.ownerID, cart.OwnerID)
			require.Equal(t, 1, len(cart.Items))
			assertCartItem(t, tt.item, cart.Items[0])
		})
	}

	// Test upsert behavior
	suite.Run("upsert existing item", func() {
		t := suite.T()
		ctx := t.Context()

		ownerID := gofakeit.UUID()
		item1 := randomCartItem()

		// Add first item
		err := suite.repo.AddItem(ctx, ownerID, item1)
		require.NoError(t, err)

		// Add same item with different price (should update)
		item2 := item1
		item2.Price = domain.Money{
			Amount:   decimal.NewFromFloat(99.99),
			Currency: item1.Price.Currency,
		}

		err = suite.repo.AddItem(ctx, ownerID, item2)
		require.NoError(t, err)

		// Verify only one item exists with updated price
		cart, err := suite.repo.GetCart(ctx, ownerID)
		require.NoError(t, err)

		require.Equal(t, 1, len(cart.Items))
		assert.Equal(t, item2.Price.Amount, cart.Items[0].Price.Amount)
	})
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		setup     func(string) error
		wantError string
		wantItems int
	}{
		{
			name:      "get empty cart: ok",
			ownerID:   gofakeit.UUID(),
			wantItems: 0,
		},
		{
			name:    "get cart with one item: ok",
			ownerID: gofakeit.UUID(),
			setup: func(ownerID string) error {
				return suite.repo.AddItem(suite.T().Context(), ownerID, randomCartItem())
			},
			wantItems: 1,
		},
		{
			name:    "get cart with multiple items: ok",
			ownerID: gofakeit.UUID(),
			setup: func(ownerID string) error {
				ctx := suite.T().Context()
				for i := 0; i < 3; i++ {
					err := suite.repo.AddItem(ctx, ownerID, randomCartItem())
					if err != nil {
						return err
					}
				}
				return nil
			},
			wantItems: 3,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			if tt.setup != nil {
				err := tt.setup(tt.ownerID)
				require.NoError(t, err)
			}

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.ownerID, cart.OwnerID)
			require.Equal(t, tt.wantItems, len(cart.Items))

			// Verify items have valid fields
			for _, item := range cart.Items {
				assert.NotEqual(t, uuid.Nil, item.ProductID)
				assert.True(t, item.Price.Amount.GreaterThan(decimal.Zero))
				assert.NotEmpty(t, item.Price.Currency.String())
				assert.False(t, item.CreatedAt.IsZero())
			}
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		productID uuid.UUID
		setup     func(string, uuid.UUID) error
		want      bool
		wantError string
	}{
		{
			name:      "delete existing item: ok",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			setup: func(ownerID string, productID uuid.UUID) error {
				item := randomCartItem()
				item.ProductID = productID
				return suite.repo.AddItem(suite.T().Context(), ownerID, item)
			},
			want: true,
		},
		{
			name:      "delete non-existing item: not found",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			want:      false,
		},
		{
			name:      "delete with empty owner ID: not found",
			ownerID:   "",
			productID: uuid.MustParse(gofakeit.UUID()),
			want:      false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			if tt.setup != nil {
				err := tt.setup(tt.ownerID, tt.productID)
				require.NoError(t, err)
			}

			deleted, err := suite.repo.DeleteItem(ctx, tt.ownerID, tt.productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.want, deleted)

			// If item was deleted, verify it's no longer in cart
			if deleted {
				cart, err := suite.repo.GetCart(ctx, tt.ownerID)
				require.NoError(t, err)

				for _, item := range cart.Items {
					assert.NotEqual(t, tt.productID, item.ProductID, "deleted item should not be in cart")
				}
			}
		})
	}
}

func (suite *cartRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE cart_items CASCADE")
	suite.NoError(err)
}

func randomCartItem() domain.CartItem {
	productID := uuid.MustParse(gofakeit.UUID())
	price := gofakeit.Price(1, 100)
	currencyUnit := randomCurrency()

	return domain.CartItem{
		ProductID: productID,
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
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

func assertCartItem(t *testing.T, expected, actual domain.CartItem) {
	t.Helper()

	currencyComparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		currencyComparer,
	}

	diff := cmp.Diff(expected, actual, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero())
}
