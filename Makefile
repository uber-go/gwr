.PHONY: lint

lint:
	go vet $$(glide novendor)

.PHONY: test

test: check-license lint
	go test $$(glide novendor)

vendor: glide.lock
	glide install

vendor/github.com/uber/uber-licence/package.json: vendor

vendor/github.com/uber/uber-licence/node_modules: vendor/github.com/uber/uber-licence/package.json
	cd vendor/github.com/uber/uber-licence && npm install

.PHONY: check-license add-license

check-license: vendor/github.com/uber/uber-licence/node_modules
	./vendor/github.com/uber/uber-licence/bin/licence --dry --file '*.go'

add-license: vendor/github.com/uber/uber-licence/node_modules
	./vendor/github.com/uber/uber-licence/bin/licence --verbose --file '*.go'
