package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	redisContainerName = "my-test-redis"
	redisHost          = "localhost"
	redisPort          = 6379
)

func prepareInfrastructure(t *testing.T, runFunc func(t *testing.T, connString string)) {
	// Start Redis container
	testRedis, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Name:  redisContainerName,
			Image: "redis:latest",
			ExposedPorts: []string{
				fmt.Sprintf("%d/tcp", redisPort),
			},
			WaitingFor: wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)
	defer cleanUp(t, testRedis.Terminate)

	redisPort, err := testRedis.MappedPort(context.Background(), nat.Port(fmt.Sprintf("%d/tcp", redisPort)))
	require.NoError(t, err)
	connString := fmt.Sprintf("%s:%d", redisHost, redisPort.Int())
	// Run tests
	runFunc(t, connString)
}

func cleanUp(t *testing.T, cleanUpFunc func(context.Context) error) {
	require.NoError(t, cleanUpFunc(context.Background()))
}
