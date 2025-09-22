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
		name       string
		ownerID    string
		itemFn     func() domain.CartItem
		testUpsert bool
		wantError  string
	}{
		{
			name:    "add valid cart item: ok",
			ownerID: gofakeit.UUID(),
			itemFn:  randomCartItem,
		},
		{
			name:       "add same item twice (upsert): ok",
			ownerID:    gofakeit.UUID(),
			itemFn:     randomCartItem,
			testUpsert: true,
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

			if tt.testUpsert {
				err = suite.repo.AddItem(ctx, tt.ownerID, item)
				require.NoError(t, err)
			}

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			assert.Equal(t, tt.ownerID, cart.OwnerID)
			assert.Len(t, cart.Items, 1)
			assertCartItem(t, item, cart.Items[0])
		})
	}
}

func (suite *cartRepositorySuite) TestGetCart() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		setupFn   func(string) []domain.CartItem
		wantError string
	}{
		{
			name:    "get empty cart: ok",
			ownerID: gofakeit.UUID(),
			setupFn: func(ownerID string) []domain.CartItem {
				return nil
			},
		},
		{
			name:    "get cart with items: ok",
			ownerID: gofakeit.UUID(),
			setupFn: func(ownerID string) []domain.CartItem {
				items := []domain.CartItem{randomCartItem(), randomCartItem()}
				for _, item := range items {
					err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
					suite.NoError(err)
				}
				return items
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			expectedItems := tt.setupFn(tt.ownerID)

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

func (suite *cartRepositorySuite) TestDeleteItem() {
	defer suite.deleteAll()

	tests := []struct {
		name      string
		ownerID   string
		setupFn   func(string) domain.CartItem
		productID func(domain.CartItem) uuid.UUID
		wantFound bool
		wantError string
	}{
		{
			name:    "delete existing item: ok",
			ownerID: gofakeit.UUID(),
			setupFn: func(ownerID string) domain.CartItem {
				item := randomCartItem()
				err := suite.repo.AddItem(suite.T().Context(), ownerID, item)
				suite.NoError(err)
				return item
			},
			productID: func(item domain.CartItem) uuid.UUID {
				return item.ProductID
			},
			wantFound: true,
		},
		{
			name:    "delete non-existing item: not found",
			ownerID: gofakeit.UUID(),
			setupFn: func(ownerID string) domain.CartItem {
				return randomCartItem()
			},
			productID: func(item domain.CartItem) uuid.UUID {
				return uuid.MustParse(gofakeit.UUID())
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			t := suite.T()
			ctx := t.Context()

			item := tt.setupFn(tt.ownerID)
			productID := tt.productID(item)

			found, err := suite.repo.DeleteItem(ctx, tt.ownerID, productID)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantFound, found)

			cart, err := suite.repo.GetCart(ctx, tt.ownerID)
			require.NoError(t, err)

			if tt.wantFound {
				assert.Len(t, cart.Items, 0)
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
