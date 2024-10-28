package reflection_test

import (
	"context"
	"testing"

	"github.com/liftedinit/yaci/internal/reflection"
	"github.com/liftedinit/yaci/internal/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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
