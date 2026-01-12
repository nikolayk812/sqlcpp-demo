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
			name:    "add item to cart: ok",
			ownerID: gofakeit.UUID(),
			item:    randomCartItem(),
		},
		{
			name:      "add item with empty owner ID: error",
			ownerID:   "",
			item:      randomCartItem(),
			wantError: "ownerID is empty",
		},
		{
			name:    "add item with zero price amount: ok",
			ownerID: gofakeit.UUID(),
			item: domain.CartItem{
				ProductID: uuid.MustParse(gofakeit.UUID()),
				Price: domain.Money{
					Amount:   decimal.Zero,
					Currency: randomCurrency(),
				},
			},
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

			require.Len(t, cart.Items, 1)
			assertCartItem(t, tt.item, cart.Items[0])
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name        string
		ownerID     string
		productID   uuid.UUID
		setupItems  []domain.CartItem
		wantDeleted bool
		wantError   string
	}{
		{
			name:      "delete existing item: ok",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			setupItems: []domain.CartItem{
				{
					ProductID: uuid.MustParse(gofakeit.UUID()),
					Price:     randomMoney(),
				},
			},
			wantDeleted: true,
		},
		{
			name:      "delete non-existing item: not found",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			setupItems: []domain.CartItem{
				{
					ProductID: uuid.MustParse(gofakeit.UUID()),
					Price:     randomMoney(),
				},
			},
			wantDeleted: false,
		},
		{
			name:        "delete from empty cart: not found",
			ownerID:     gofakeit.UUID(),
			productID:   uuid.MustParse(gofakeit.UUID()),
			setupItems:  []domain.CartItem{},
			wantDeleted: false,
		},
		{
			name:      "delete with empty owner ID: error",
			ownerID:   "",
			productID: uuid.MustParse(gofakeit.UUID()),
			wantError: "ownerID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// Setup: add items to cart
			for i, item := range tt.setupItems {
				// Use the productID from test case for the first item in "delete existing" test
				if tt.name == "delete existing item: ok" && i == 0 {
					item.ProductID = tt.productID
				}
				err := suite.repo.AddItem(ctx, tt.ownerID, item)
				require.NoError(t, err)
			}

			// Test the deletion
			deleted, err := suite.repo.DeleteItem(ctx, tt.ownerID, tt.productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDeleted, deleted)
		})
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name       string
		ownerID    string
		setupItems []domain.CartItem
		wantError  string
	}{
		{
			name:    "get cart with items: ok",
			ownerID: gofakeit.UUID(),
			setupItems: []domain.CartItem{
				randomCartItem(),
				randomCartItem(),
			},
		},
		{
			name:       "get empty cart: ok",
			ownerID:    gofakeit.UUID(),
			setupItems: []domain.CartItem{},
		},
		{
			name:      "get cart with empty owner ID: error",
			ownerID:   "",
			wantError: "ownerID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			// Setup: add items to cart
			for _, item := range tt.setupItems {
				err := suite.repo.AddItem(ctx, tt.ownerID, item)
				require.NoError(t, err)
			}

			// Test getting the cart
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.ownerID, cart.OwnerID)
			assert.Len(t, cart.Items, len(tt.setupItems))

			// Verify each item
			for i, expectedItem := range tt.setupItems {
				assertCartItem(t, expectedItem, cart.Items[i])
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
		Price:     randomMoney(),
	}
}

func randomMoney() domain.Money {
	return domain.Money{
		Amount:   decimal.NewFromFloat(gofakeit.Price(1, 100)),
		Currency: randomCurrency(),
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

	// Ignore the CreatedAt field in CartItem
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		currencyComparer,
	}

	diff := cmp.Diff(expected, actual, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero())
}
