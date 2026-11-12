# Пример gRPC-клиента и сервера на Go

Минимальный сервис `Greeter`: клиент отправляет имя, сервер отвечает приветствием.
Контракт описан в Protocol Buffers, Go-код клиента и сервера сгенерирован из `.proto`.

## Структура

- `api/greeter/v1/greeter.proto` — protobuf-контракт.
- `gen/greeter/v1/` — сгенерированные protobuf-типы и gRPC-клиент/сервер.
- `cmd/server/` — gRPC-сервер на порту `50051`.
- `cmd/client/` — клиент, принимающий имя аргументом командной строки.

## Запуск

Из каталога `lecture-2-1` откройте два терминала.

В первом запустите сервер:

```bash
go run ./cmd/server
```

Во втором отправьте запрос:

```bash
go run ./cmd/client Юрий
```

Ожидаемый ответ:

```text
Hello, Юрий!
```

Если имя не указано, клиент использует `gRPC`.

## Генерация protobuf-кода

Для генерации нужны компилятор `protoc` и Go-плагины `protoc-gen-go` и
`protoc-gen-go-grpc`. На Ubuntu установите компилятор:

```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

Проверьте, что компилятор доступен:

```bash
protoc --version
```

Установите Go-плагины:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
```

Добавьте каталог с Go-плагинами в `PATH` (для текущего терминала):

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Проверьте, что команда видит все три инструмента:

```bash
command -v protoc
command -v protoc-gen-go
command -v protoc-gen-go-grpc
```

Затем из каталога `lecture-2-1` выполните:

```bash
make generate
```

Команда генерации описана в `Makefile`. После изменения `.proto` сгенерируйте код заново.
