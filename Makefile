export CGO_ENABLED = 0
export NEXT_TELEMETRY_DISABLED = 1

.PHONY: build
build: build-admin
	go build ./cmd/proxima

.PHONY: build-admin
build-admin:
	cd admin && \
	yarn install --frozen-lockfile && \
	yarn run export && \
	mv dist ../cmd/proxima/admin

.PHONY: clean
clean:
	rm -f proxima
	rm -rf ./cmd/proxima/admin
	rm -rf ./admin/dist
	rm -rf ./admin/.next
