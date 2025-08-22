package repository_test

import (
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
	"github.com/testcontainers/testcontainers-go"
	"go.uber.org/goleak"
	"golang.org/x/text/currency"
	"os"
	"sort"
	"testing"
	"time"
)

type cartRepositorySuite struct {
	suite.Suite

	pool      *pgxpool.Pool
	repo      port.CartRepository
	container testcontainers.Container
}

// entry point to run the tests in the suite
func TestCartRepositorySuite(t *testing.T) {
	require.NoError(t, os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"))

	// Verifies no leaks after all tests in the suite run.
	defer goleak.VerifyNone(t)

	suite.Run(t, new(cartRepositorySuite))
}

// before all tests in the suite
func (suite *cartRepositorySuite) SetupSuite() {
	ctx := suite.T().Context()

	var (
		connStr string
		err     error
	)

	suite.container, connStr, err = startPostgres(ctx)
	suite.NoError(err)

	suite.pool, err = pgxpool.New(ctx, connStr)
	suite.NoError(err)

	suite.repo = repository.NewCart(suite.pool)
	suite.NoError(err)
}

// after all tests in the suite
func (suite *cartRepositorySuite) TearDownSuite() {
	ctx := suite.T().Context()

	if suite.pool != nil {
		suite.pool.Close()
	}
	if suite.container != nil {
		suite.NoError(suite.container.Terminate(ctx))
	}
}

func (suite *cartRepositorySuite) TestAddItem() {
	defer suite.deleteAll()

	ownerID := gofakeit.UUID()
	item1 := fakeCartItem()
	item2 := fakeCartItem()

	tests := []struct {
		name      string
		ownerID   string
		item      domain.CartItem
		wantError string
	}{
		{
			name:    "add item to cart: ok",
			ownerID: ownerID,
			item:    item1,
		},
		{
			name:    "add another item to cart: ok",
			ownerID: ownerID,
			item:    item2,
		},
		{
			name:    "add item with same product ID (upsert): ok",
			ownerID: ownerID,
			item:    item1,
		},
		{
			name:      "add item with empty owner ID: error",
			ownerID:   "",
			item:      item1,
			wantError: "ownerID is empty",
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

			// Verify the item was added by getting the cart
			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			// Find the item in the cart
			found := false
			for _, cartItem := range cart.Items {
				if cartItem.ProductID == tt.item.ProductID {
					found = true
					assertCartItem(t, tt.item, cartItem)
					break
				}
			}
			assert.True(t, found, "Item should be found in cart")
		})
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	ownerID1 := gofakeit.UUID()
	ownerID2 := gofakeit.UUID()
	item1 := fakeCartItem()
	item2 := fakeCartItem()

	// Add items to different carts
	err := suite.repo.AddItem(suite.T().Context(), ownerID1, item1)
	suite.NoError(err)
	err = suite.repo.AddItem(suite.T().Context(), ownerID1, item2)
	suite.NoError(err)
	err = suite.repo.AddItem(suite.T().Context(), ownerID2, item1)
	suite.NoError(err)

	tests := []struct {
		name        string
		ownerID     string
		wantItems   []domain.CartItem
		wantError   string
	}{
		{
			name:      "get cart with items: ok",
			ownerID:   ownerID1,
			wantItems: []domain.CartItem{item1, item2},
		},
		{
			name:      "get cart with single item: ok",
			ownerID:   ownerID2,
			wantItems: []domain.CartItem{item1},
		},
		{
			name:    "get empty cart: ok",
			ownerID: gofakeit.UUID(),
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

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.ownerID, cart.OwnerID)
			assertCartItems(t, tt.wantItems, cart.Items)
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	ownerID := gofakeit.UUID()
	item1 := fakeCartItem()
	item2 := fakeCartItem()

	// Add items to cart
	err := suite.repo.AddItem(suite.T().Context(), ownerID, item1)
	suite.NoError(err)
	err = suite.repo.AddItem(suite.T().Context(), ownerID, item2)
	suite.NoError(err)

	tests := []struct {
		name      string
		ownerID   string
		productID uuid.UUID
		wantFound bool
		wantError string
	}{
		{
			name:      "delete existing item: ok",
			ownerID:   ownerID,
			productID: item1.ProductID,
			wantFound: true,
		},
		{
			name:      "delete non-existing item: not found",
			ownerID:   ownerID,
			productID: uuid.MustParse(gofakeit.UUID()),
			wantFound: false,
		},
		{
			name:      "delete item with empty owner ID: error",
			ownerID:   "",
			productID: item1.ProductID,
			wantError: "ownerID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			found, err := suite.repo.DeleteItem(ctx, tt.ownerID, tt.productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				// Verify the item was deleted by getting the cart
				cart, err := suite.repo.GetCart(ctx, tt.ownerID)
				require.NoError(t, err)

				// Verify the item is not in the cart
				for _, cartItem := range cart.Items {
					assert.NotEqual(t, tt.productID, cartItem.ProductID, "Deleted item should not be in cart")
				}
			}
		})
	}
}

func (suite *cartRepositorySuite) deleteAll() {
	_, err := suite.pool.Exec(suite.T().Context(), "TRUNCATE TABLE cart_items CASCADE")
	suite.NoError(err)
}

func fakeCartItem() domain.CartItem {
	productID := uuid.MustParse(gofakeit.UUID())
	price := gofakeit.Price(1, 100)
	currencyUnit := fakeCurrency()

	return domain.CartItem{
		ProductID: productID,
		Price: domain.Money{
			Amount:   decimal.NewFromFloat(price),
			Currency: currencyUnit,
		},
		CreatedAt: time.Now().UTC(),
	}
}

func fakeCurrency() currency.Unit {
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

func assertCartItem(t *testing.T, expected domain.CartItem, actual domain.CartItem) {
	t.Helper()

	currencyComparer := cmp.Comparer(func(x, y currency.Unit) bool {
		return x.String() == y.String()
	})

	// Ignore the CreatedAt field
	opts := cmp.Options{
		cmpopts.IgnoreFields(domain.CartItem{}, "CreatedAt"),
		currencyComparer,
	}

	diff := cmp.Diff(expected, actual, opts)
	assert.Empty(t, diff)

	assert.False(t, actual.CreatedAt.IsZero())
}

func assertCartItems(t *testing.T, expected []domain.CartItem, actual []domain.CartItem) {
	t.Helper()

	sortCartItems := func(items []domain.CartItem) {
		sort.Slice(items, func(i, j int) bool {
			return items[i].ProductID.String() < items[j].ProductID.String()
		})
	}

	sortCartItems(expected)
	sortCartItems(actual)

	require.Equal(t, len(expected), len(actual))

	for i := range expected {
		assertCartItem(t, expected[i], actual[i])
	}
}