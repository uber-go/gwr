.PHONY: lint

lint:
	go vet $$(glide novendor)

.PHONY: test

test: lint
	go test $$(glide novendor)
