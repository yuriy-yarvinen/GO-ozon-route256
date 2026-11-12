MODULE := gitlab.ozon.dev/go/classroom-8/students/lecture-2-1
PROTO := api/greeter/v1/greeter.proto

.PHONY: generate
generate:
	@command -v protoc >/dev/null 2>&1 || { echo "protoc не найден. Установите его: sudo apt install protobuf-compiler" >&2; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go не найден. Установите Go-плагины и добавьте GOPATH/bin в PATH (см. README.md)" >&2; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc не найден. Установите Go-плагины и добавьте GOPATH/bin в PATH (см. README.md)" >&2; exit 1; }
	protoc -I . \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		$(PROTO)
