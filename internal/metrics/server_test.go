package metrics_test

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/manifest-network/yaci/internal/metrics"
	"github.com/manifest-network/yaci/internal/metrics/collectors"
	"github.com/stretchr/testify/require"
)

func TestCreateMetricsServer(t *testing.T) {
	t.Run("StartServer", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectQuery(regexp.QuoteMeta(collectors.TotalTransactionCountQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(28))
		mock.ExpectQuery(regexp.QuoteMeta(collectors.TotalUniqueAddressesQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"user_count", "group_count"}).AddRow(2, 2))

		server, err := metrics.CreateMetricsServer(db, "manifest", "127.0.0.1:2112")
		require.NoError(t, err)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			require.NoError(t, err)
		}()

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://127.0.0.1:2112/metrics")
		require.NoError(t, err, "Failed to connect to metrics server")

		// Read and log response body if status isn't 200
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Error response: %s", string(body))
		}
		require.Equal(t, 200, resp.StatusCode, "Expected status code 200")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("WhenInvalidAddress", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		_, err = metrics.CreateMetricsServer(db, "manifest", "invalid-address😆")
		require.Error(t, err)
	})

	t.Run("WhenInvalidPort", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		_, err = metrics.CreateMetricsServer(db, "manifest", "localhost:99999")
		require.Error(t, err)
	})

	t.Run("ValidPort", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		server, err := metrics.CreateMetricsServer(db, "manifest", "localhost:12345")
		require.NoError(t, err)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			require.NoError(t, err)
		}()
	})
}
