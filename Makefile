export CGO_ENABLED = 0

.PHONY: build-admin build clean

# Build the React/Vite admin UI and stage it where the Go binary embeds it
# (cmd/proxima/admin/dist, matching the //go:embed directive in proxima.go).
build-admin:
	cd admin && npm ci && npm run build
	rm -rf cmd/proxima/admin
	mkdir -p cmd/proxima/admin
	cp -r admin/dist cmd/proxima/admin/dist

build: build-admin
	go build -o proxima ./cmd/proxima

clean:
	rm -f proxima
	rm -rf cmd/proxima/admin
	rm -rf admin/dist
