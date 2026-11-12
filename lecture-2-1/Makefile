.PHONY: generate
generate:
	@command -v buf >/dev/null 2>&1 || { echo "buf не найден. Установите Buf CLI (см. README.md)" >&2; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go не найден. Установите Go-плагины и добавьте GOPATH/bin в PATH (см. README.md)" >&2; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc не найден. Установите Go-плагины и добавьте GOPATH/bin в PATH (см. README.md)" >&2; exit 1; }
	buf generate
