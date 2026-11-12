# Пример gRPC-клиента и сервера на Go

Минимальный сервис `Greeter`: клиент отправляет имя, сервер отвечает приветствием.
Контракт описан в Protocol Buffers, Go-код клиента и сервера сгенерирован из `.proto`.

## Структура

- `api/greeter/v1/greeter.proto` — protobuf-контракт.
- `api/examples/` — отдельные примеры синтаксиса и встроенных типов Protocol Buffers.
- `gen/greeter/v1/` — сгенерированные protobuf-типы и gRPC-клиент/сервер.
- `cmd/server/` — gRPC-сервер на порту `50051`.
- `cmd/client/` — клиент, принимающий имя аргументом командной строки.

## Примеры protobuf

В `api/examples/` собраны небольшие контракты по отдельным темам:

- `deprecated.proto` — пометка устаревших полей, enum и методов.
- `enum.proto` — перечисления.
- `google_common_types.proto` — распространённые типы из `google/type`.
- `google_well_known_types.proto` — встроенные типы `google/protobuf`, например `Timestamp`, `Duration` и `FieldMask`.
- `maps.proto` — поля-карты.
- `one_of.proto` — выбор одного значения из нескольких вариантов.
- `optional.proto` — явное присутствие необязательных полей в proto3.
- `scalar_value_types.proto` — скалярные типы protobuf.

Файлы в `api/examples/` предназначены для изучения и не включены в `make generate`,
который генерирует код для gRPC-сервиса `Greeter`. Для компиляции
`google_well_known_types.proto` укажите `protoc` каталог с установленными
protobuf include-файлами (`google/protobuf`). Для `google_common_types.proto`
также нужны `.proto`-файлы Google APIs (`google/type`). Зависимость описана в
`buf.yaml`. Для загрузки её в локальный кэш Buf выполните `buf dep update` из
каталога `lecture-2-1`; это также создаст `buf.lock`, который использует Buf LSP.
Корневой `buf.work.yaml` подключает этот модуль к Buf workspace, чтобы Buf LSP
находил зависимость и при открытом в редакторе корне репозитория.

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
