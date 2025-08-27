build:
	@echo building...
	@go build -o ./bin/api ./cmd/api
	@echo done

run:build
	@./bin/api
