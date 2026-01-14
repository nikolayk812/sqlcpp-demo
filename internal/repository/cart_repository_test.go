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
		ownerID   string
		items     []domain.CartItem
		wantError string
	}{
		{
			name:    "empty cart: ok",
			ownerID: gofakeit.UUID(),
		},
		{
			name:    "cart with single item: ok",
			ownerID: gofakeit.UUID(),
			items:   []domain.CartItem{randomCartItem()},
		},
		{
			name:    "cart with multiple items: ok",
			ownerID: gofakeit.UUID(),
			items: []domain.CartItem{
				randomCartItem(),
				randomCartItem(),
				randomCartItem(),
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// Add items to cart if any
			for _, item := range tt.items {
				err := suite.repo.AddItem(ctx, tt.ownerID, item)
				require.NoError(t, err)
			}

			actualCart, err := suite.repo.GetCart(ctx, tt.ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			expected := domain.Cart{
				OwnerID: tt.ownerID,
				Items:   tt.items,
			}

			assertCart(t, expected, actualCart)
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
			name:    "add item to empty cart: ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
		{
			name:    "add item with same product ID (upsert): ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// For upsert test, first add the item
			if tt.name == "add item with same product ID (upsert): ok" {
				firstItem := tt.item
				firstItem.Price = domain.Money{
					Amount:   decimal.NewFromFloat(100.0),
					Currency: currency.USD,
				}
				err := suite.repo.AddItem(ctx, tt.ownerID, firstItem)
				require.NoError(t, err)
			}

			err := suite.repo.AddItem(ctx, tt.ownerID, tt.item)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify item was added/updated
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)
			require.Len(t, cart.Items, 1)

			// For upsert test, verify the price was updated
			actualItem := cart.Items[0]
			require.Equal(t, tt.item.ProductID, actualItem.ProductID)
			require.Equal(t, tt.item.Price.Amount.String(), actualItem.Price.Amount.String())
			require.Equal(t, tt.item.Price.Currency, actualItem.Price.Currency)
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name        string
		ownerID     string
		setupItems  []domain.CartItem
		deleteID    uuid.UUID
		wantDeleted bool
		wantError   string
	}{
		{
			name:        "delete existing item: ok",
			ownerID:     gofakeit.UUID(),
			setupItems:  []domain.CartItem{randomCartItem()},
			wantDeleted: true,
		},
		{
			name:        "delete non-existing item: false",
			ownerID:     gofakeit.UUID(),
			setupItems:  []domain.CartItem{randomCartItem()},
			deleteID:    uuid.MustParse(gofakeit.UUID()),
			wantDeleted: false,
		},
		{
			name:        "delete from empty cart: false",
			ownerID:     gofakeit.UUID(),
			deleteID:    uuid.MustParse(gofakeit.UUID()),
			wantDeleted: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// Add setup items
			for _, item := range tt.setupItems {
				err := suite.repo.AddItem(ctx, tt.ownerID, item)
				require.NoError(t, err)
			}

			deleteID := tt.deleteID
			if deleteID == uuid.Nil && len(tt.setupItems) > 0 {
				deleteID = tt.setupItems[0].ProductID
			}

			deleted, err := suite.repo.DeleteItem(ctx, tt.ownerID, deleteID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantDeleted, deleted)

			// Verify item was deleted if expected
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			if tt.wantDeleted {
				// Item should be removed
				for _, item := range cart.Items {
					require.NotEqual(t, deleteID, item.ProductID)
				}
			} else if len(tt.setupItems) > 0 {
				// Items should still be there
				require.Len(t, cart.Items, len(tt.setupItems))
			}
		})
	}
}

func (suite *cartRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE cart_items CASCADE")
	suite.NoError(err)
}

func randomCartItem() domain.CartItem {
	return domain.CartItem{
		ProductID: uuid.MustParse(gofakeit.UUID()),
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(gofakeit.Price(1, 100)),
			Currency: randomCurrency(),
		},
	}
}

func randomCurrency() currency.Unit {
	var (
		result currency.Unit
		err    error
	)

	for {
		result, err = currency.ParseISO(gofakeit.CurrencyShort())
		if err == nil {
			break
		}
	}

	return result
}

func assertCart(t *testing.T, expected, actual domain.Cart) {
	t.Helper()

	currencyComparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	// Ignore the CreatedAt field in CartItem
	// Treat empty slices as equal to nil
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		currencyComparer,
		cmpopts.SortSlices(func(x, y domain.CartItem) bool {
			return x.ProductID.String() < y.ProductID.String()
		}),
		cmpopts.EquateEmpty(),
	}

	diff := cmp.Diff(expected, actual, opts)
	require.Empty(t, diff)
}
