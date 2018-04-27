TMP_REDIS_CONTAINER_NAME=worker-redis-tests

.PHONY: tests
tests:
	-docker rm -f $(TMP_REDIS_CONTAINER_NAME)
	-docker run -d -p 6379:6379 --name $(TMP_REDIS_CONTAINER_NAME) redis
	GOCACHE=off go test -race -v ./...
	-docker rm -f $(TMP_REDIS_CONTAINER_NAME)
