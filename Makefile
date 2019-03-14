dep:
	dep ensure

vet:
	go vet ./...

test: dep vet
	go test -race -cover ./...

fmt:
	go fmt ./...
