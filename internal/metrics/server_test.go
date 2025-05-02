package metrics_test

import (
	"context"
	"log"
	"net"
	"net/http"
	"regexp"
	"testing"
	"time"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/liftedinit/yaci/internal/client"
	"github.com/liftedinit/yaci/internal/metrics"
	"github.com/liftedinit/yaci/internal/metrics/collectors/sql"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// mockBankServer is a mock implementation of the bankv1beta1.QueryServer interface.
type mockBankServer struct {
	bankv1beta1.UnimplementedQueryServer
}

func (s *mockBankServer) DenomsMetadata(_ context.Context, _ *bankv1beta1.QueryDenomsMetadataRequest) (*bankv1beta1.QueryDenomsMetadataResponse, error) {
	return &bankv1beta1.QueryDenomsMetadataResponse{Pagination: &queryv1beta1.PageResponse{Total: 1}}, nil
}

func (s *mockBankServer) SupplyOf(_ context.Context, _ *bankv1beta1.QuerySupplyOfRequest) (*bankv1beta1.QuerySupplyOfResponse, error) {
	return &bankv1beta1.QuerySupplyOfResponse{}, nil
}

func (s *mockBankServer) DenomMetadata(_ context.Context, _ *bankv1beta1.QueryDenomMetadataRequest) (*bankv1beta1.QueryDenomMetadataResponse, error) {
	return &bankv1beta1.QueryDenomMetadataResponse{}, nil
}

// setupMockGrpcServer sets up a mock gRPC server for testing.
func setupMockGrpcServer() *grpc.Server {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	// Register your mock service implementation
	bankv1beta1.RegisterQueryServer(s, &mockBankServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			// Use t.Logf or similar in actual test setup
			log.Printf("Mock Server exited with error: %v", err)
		}
	}()
	return s
}

// bufDialer is a custom dialer for the bufconn listener.
func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

// setupMockGrpcClient sets up a mock gRPC client for testing.
func setupMockGrpcClient(t *testing.T) *client.GRPCClient {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Use insecure for testing
	)
	require.NoError(t, err)
	return &client.GRPCClient{
		Ctx:  ctx,
		Conn: conn,
	}
}

// waitForServerReady waits for the server to be ready by attempting to connect to it.
func waitForServerReady(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("Server at %s did not become ready within %v", addr, timeout)
}

func TestCreateMetricsServer(t *testing.T) {
	// SQL mock setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// gRPC mock setup
	mockServer := setupMockGrpcServer()
	defer mockServer.Stop()
	mockClient := setupMockGrpcClient(t)
	defer mockClient.Conn.Close()

	t.Run("StartServer", func(t *testing.T) {
		server, err := metrics.CreateMetricsServer(db, mockClient, "manifest", "127.0.0.1:2112")
		require.NoError(t, err)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			require.NoError(t, err)
		}()

		waitForServerReady(t, server.Addr, 1*time.Second)
	})

	t.Run("DBCollectors", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(sql.TotalTransactionCountQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(28))
		mock.ExpectQuery(regexp.QuoteMeta(sql.TotalUniqueAddressesQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"user_count", "group_count"}).AddRow(2, 2))

		server, err := metrics.CreateMetricsServer(db, mockClient, "manifest", "127.0.0.1:2112")
		require.NoError(t, err)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			require.NoError(t, err)
		}()

		waitForServerReady(t, server.Addr, 1*time.Second)

		_, err = http.Get("http://127.0.0.1:2112/metrics")
		require.NoError(t, err, "Failed to connect to metrics server")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GRPCCollectors", func(t *testing.T) {
		t.Skip("TODO: Implement gRPC collector test")
	})

	t.Run("WhenInvalidAddress", func(t *testing.T) {
		_, err = metrics.CreateMetricsServer(db, mockClient, "manifest", "invalid-address😆")
		require.Error(t, err)
	})

	t.Run("WhenInvalidPort", func(t *testing.T) {
		_, err = metrics.CreateMetricsServer(db, mockClient, "manifest", "localhost:99999")
		require.Error(t, err)
	})

	t.Run("ValidPort", func(t *testing.T) {
		server, err := metrics.CreateMetricsServer(db, mockClient, "manifest", "localhost:12345")
		require.NoError(t, err)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			require.NoError(t, err)
		}()
	})
}
