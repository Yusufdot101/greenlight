build:
	@echo building...
	@go build -o ./temp/api ./cmd/api
	@echo done.
run:build
	@./temp/api

