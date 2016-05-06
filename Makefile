.PHONY: lint

lint:
	go vet $$(glide novendor)

test: lint
	go test $$(glide novendor)
