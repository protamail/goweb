all: test

test:
	go fmt
	go test

bundle: esbuild/bundle.go
	go build esbuild/bundle.go

watchAndReload: reloader/watchAndReload.go
	go build reloader/watchAndReload.go
