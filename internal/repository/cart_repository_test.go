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
	"golang.org/x/text/currency"
	"sort"
	"testing"
	"time"
)

type cartRepositorySuite struct {
	suite.Suite

	repo port.CartRepository
	pool *pgxpool.Pool
}

func TestCartRepositorySuite(t *testing.T) {
	suite.Run(t, new(cartRepositorySuite))
}

func (suite *cartRepositorySuite) SetupSuite() {
	ctx := suite.T().Context()

	_, connStr, err := startPostgres(ctx)
	suite.NoError(err)

	suite.pool, err = pgxpool.New(ctx, connStr)
	suite.NoError(err)

	suite.repo, err = repository.NewCart(suite.pool)
	suite.NoError(err)
}

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
		itemFn    func() domain.CartItem
		wantError string
	}{
		{
			name:    "add valid item: ok",
			ownerID: gofakeit.UUID(),
			itemFn:  randomCartItem,
		},
		{
			name:      "add item with empty owner ID: error",
			ownerID:   "",
			itemFn:    randomCartItem,
			wantError: "ownerID is empty",
		},
		{
			name:    "add item, update existing product: ok",
			ownerID: gofakeit.UUID(),
			itemFn:  randomCartItem,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			item := tt.itemFn()

			err := suite.repo.AddItem(ctx, tt.ownerID, item)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			assert.Equal(t, tt.ownerID, cart.OwnerID)
			require.Len(t, cart.Items, 1)
			assertCartItem(t, item, cart.Items[0])
		})
	}
}

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		productID uuid.UUID
		setup     func(string, domain.CartItem)
		wantFound bool
		wantError string
	}{
		{
			name:      "delete existing item: ok",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			setup: func(ownerID string, item domain.CartItem) {
				err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
				suite.NoError(err)
			},
			wantFound: true,
		},
		{
			name:      "delete non-existing item: not found",
			ownerID:   gofakeit.UUID(),
			productID: uuid.MustParse(gofakeit.UUID()),
			wantFound: false,
		},
		{
			name:      "delete with empty owner ID: error",
			ownerID:   "",
			productID: uuid.MustParse(gofakeit.UUID()),
			wantError: "ownerID is empty",
		},
		{
			name:      "delete with empty product ID: error",
			ownerID:   gofakeit.UUID(),
			productID: uuid.Nil,
			wantError: "productID is empty",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			if tt.setup != nil {
				item := randomCartItem()
				item.ProductID = tt.productID
				tt.setup(tt.ownerID, item)
			}

			found, err := suite.repo.DeleteItem(ctx, tt.ownerID, tt.productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				cart, err := suite.repo.GetCart(ctx, tt.ownerID)
				require.NoError(t, err)
				assert.Empty(t, cart.Items)
			}
		})
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		setup     func(string) []domain.CartItem
		wantError string
	}{
		{
			name:    "get empty cart: ok",
			ownerID: gofakeit.UUID(),
		},
		{
			name:    "get cart with items: ok",
			ownerID: gofakeit.UUID(),
			setup: func(ownerID string) []domain.CartItem {
				items := []domain.CartItem{randomCartItem(), randomCartItem()}
				for _, item := range items {
					err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
					suite.NoError(err)
				}
				return items
			},
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

			var expectedItems []domain.CartItem
			if tt.setup != nil {
				expectedItems = tt.setup(tt.ownerID)
			}

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.ownerID, cart.OwnerID)
			assert.Len(t, cart.Items, len(expectedItems))

			if len(expectedItems) > 0 {
				assertCartItems(t, expectedItems, cart.Items)
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
		CreatedAt: time.Now().UTC(),
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

func assertCartItem(t *testing.T, expected domain.CartItem, actual domain.CartItem) {
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
