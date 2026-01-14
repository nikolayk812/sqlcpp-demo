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
		itemFunc  func() (string, domain.CartItem)
		wantError string
	}{
		{
			name:     "valid cart item: ok",
			itemFunc: randomCartItem,
		},
		{
			name: "empty owner ID: error",
			itemFunc: func() (string, domain.CartItem) {
				_, item := randomCartItem()
				return "", item
			},
			wantError: "ownerID is empty",
		},
		{
			name: "nil product ID: error",
			itemFunc: func() (string, domain.CartItem) {
				ownerID, item := randomCartItem()
				item.ProductID = uuid.Nil
				return ownerID, item
			},
			wantError: "item.ProductID is empty",
		},
		{
			name: "zero price amount: ok",
			itemFunc: func() (string, domain.CartItem) {
				ownerID, item := randomCartItem()
				item.Price.Amount = decimal.Zero
				return ownerID, item
			},
		},
		{
			name: "upsert existing item: ok",
			itemFunc: func() (string, domain.CartItem) {
				ownerID, item := randomCartItem()
				// First, add the item
				ctx := suite.T().Context()
				err := suite.repo.AddItem(ctx, ownerID, item)
				suite.NoError(err)

				// Now return the same item with different price for upsert
				item.Price.Amount = decimal.NewFromFloat(99.99)
				return ownerID, item
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			ownerID, ttItem := tt.itemFunc()

			err := suite.repo.AddItem(ctx, ownerID, ttItem)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			// Verify the item was added by getting the cart
			actualCart, err := suite.repo.GetCart(ctx, ownerID)
			require.NoError(t, err)

			require.Len(t, actualCart.Items, 1)
			assertCartItem(t, ttItem, actualCart.Items[0])
		})
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		setupFunc func() (string, []domain.CartItem)
		wantError string
	}{
		{
			name: "cart with multiple items: ok",
			setupFunc: func() (string, []domain.CartItem) {
				ownerID := gofakeit.UUID()
				items := []domain.CartItem{
					randomCartItemForOwner(ownerID).item,
					randomCartItemForOwner(ownerID).item,
					randomCartItemForOwner(ownerID).item,
				}

				ctx := suite.T().Context()
				for _, item := range items {
					err := suite.repo.AddItem(ctx, ownerID, item)
					suite.NoError(err)
				}

				return ownerID, items
			},
		},
		{
			name: "empty cart: ok",
			setupFunc: func() (string, []domain.CartItem) {
				return gofakeit.UUID(), nil
			},
		},
		{
			name: "empty owner ID: error",
			setupFunc: func() (string, []domain.CartItem) {
				return "", nil
			},
			wantError: "ownerID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			ownerID, expectedItems := tt.setupFunc()

			actualCart, err := suite.repo.GetCart(ctx, ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, ownerID, actualCart.OwnerID)
			require.Len(t, actualCart.Items, len(expectedItems))

			// Sort items by ProductID for consistent comparison
			expectedMap := make(map[uuid.UUID]domain.CartItem)
			for _, item := range expectedItems {
				expectedMap[item.ProductID] = item
			}

			for _, actualItem := range actualCart.Items {
				expectedItem, exists := expectedMap[actualItem.ProductID]
				require.True(t, exists, "unexpected item in cart: %v", actualItem.ProductID)
				assertCartItem(t, expectedItem, actualItem)
			}
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name        string
		setupFunc   func() (string, uuid.UUID, bool) // returns ownerID, productID, expectSuccess
		wantError   string
		wantDeleted bool
	}{
		{
			name: "delete existing item: ok",
			setupFunc: func() (string, uuid.UUID, bool) {
				ownerID, item := randomCartItem()
				ctx := suite.T().Context()
				err := suite.repo.AddItem(ctx, ownerID, item)
				suite.NoError(err)
				return ownerID, item.ProductID, true
			},
			wantDeleted: true,
		},
		{
			name: "delete non-existing item: false",
			setupFunc: func() (string, uuid.UUID, bool) {
				ownerID := gofakeit.UUID()
				productID := uuid.MustParse(gofakeit.UUID())
				return ownerID, productID, false
			},
			wantDeleted: false,
		},
		{
			name: "empty owner ID: error",
			setupFunc: func() (string, uuid.UUID, bool) {
				productID := uuid.MustParse(gofakeit.UUID())
				return "", productID, false
			},
			wantError: "ownerID is empty",
		},
		{
			name: "nil product ID: error",
			setupFunc: func() (string, uuid.UUID, bool) {
				ownerID := gofakeit.UUID()
				return ownerID, uuid.Nil, false
			},
			wantError: "productID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			ownerID, productID, _ := tt.setupFunc()

			deleted, err := suite.repo.DeleteItem(ctx, ownerID, productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantDeleted, deleted)

			// Verify the item was actually deleted by checking the cart
			if tt.wantDeleted {
				cart, err := suite.repo.GetCart(ctx, ownerID)
				require.NoError(t, err)

				for _, item := range cart.Items {
					assert.NotEqual(t, productID, item.ProductID, "item should have been deleted")
				}
			}
		})
	}
}

func (suite *cartRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE cart_items CASCADE")
	suite.NoError(err)
}

func randomCartItem() (string, domain.CartItem) {
	ownerID := gofakeit.UUID()
	return randomCartItemForOwner(ownerID).ownerID, randomCartItemForOwner(ownerID).item
}

type cartItemPair struct {
	ownerID string
	item    domain.CartItem
}

func randomCartItemForOwner(ownerID string) cartItemPair {
	productID := uuid.MustParse(gofakeit.UUID())
	price := gofakeit.Price(1, 1000)
	currencyUnit := randomCurrency()

	return cartItemPair{
		ownerID: ownerID,
		item: domain.CartItem{
			ProductID: productID,
			Price: domain.Money{
				Amount:   decimal.NewFromFloat(price),
				Currency: currencyUnit,
			},
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

	// Ignore the CreatedAt field since it's set by the database
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		currencyComparer,
	}

	diff := cmp.Diff(expected, actual, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero(), "CreatedAt should be set")
}
