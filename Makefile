protoc:
	protoc -I ./apis --go_out=./apis --go-grpc_out=./apis ./apis/*.proto

before-push:
	go mod tidy &&\
	gofumpt -l -w . &&\
	go build ./...

run-redis:
	@docker stop my-redis || true &&\
	docker run --name my-redis --rm  -p 6379:6379  -d redis