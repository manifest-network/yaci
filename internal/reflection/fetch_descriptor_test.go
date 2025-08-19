package reflection_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/manifest-network/yaci/internal/reflection"
	"github.com/manifest-network/yaci/internal/testutil"
)

func TestFetchAllDescriptors(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(testutil.MockDialer), grpc.WithInsecure())
	assert.NoError(t, err)
	defer conn.Close()

	descriptors, err := reflection.FetchAllDescriptors(ctx, conn, 3)
	assert.NoError(t, err)
	assert.NotNil(t, descriptors)
	assert.NotEmpty(t, descriptors)
}
