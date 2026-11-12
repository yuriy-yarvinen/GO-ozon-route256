# Пример gRPC-клиента и сервера на Go

Минимальный сервис `Greeter`: клиент отправляет имя, сервер отвечает приветствием.
Контракт описан в Protocol Buffers, Go-код клиента и сервера сгенерирован из `.proto`.

## Структура

- `api/greeter/v1/greeter.proto` — protobuf-контракт.
- `api/examples/` — отдельные примеры синтаксиса и встроенных типов Protocol Buffers.
- `gen/` — сгенерированные protobuf-типы и gRPC-клиенты/серверы.
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

Клиент вызывает каждый пример через соответствующий метод `Examples` и печатает
полученный от сервера protobuf-объект. Сервер возвращает отправленные значения,
чтобы можно было проверить сериализацию и передачу каждого типа.

Файл `buf.yaml` подключает Google APIs для типов из `google/type`. Перед
генерацией кода установите Buf CLI и Go-плагины `protoc-gen-go` и
`protoc-gen-go-grpc`. Зависимости Buf загружаются командой `buf dep update` из
каталога `lecture-2-1`. Корневой `buf.work.yaml` подключает модуль к Buf
workspace для Buf LSP при открытом корне репозитория.

## Запуск

Из каталога `lecture-2-1` откройте два терминала.

В первом запустите сервер:

```bash
go run ./cmd/server
```

Во втором терминале клиент вызовет Greeter и все восемь методов примеров:

```bash
go run ./cmd/client Юрий
```

По умолчанию используется `localhost:50051`. Чтобы выбрать другой адрес, задайте
одинаковое значение `GRPC_ADDR` для сервера и клиента, например
`GRPC_ADDR=localhost:50052`.

Сначала клиент выведет приветствие, затем ответы всех методов в JSON.
Например:

```text
Hello, Юрий!
```

Если имя не указано, клиент использует `gRPC`.

## Генерация protobuf-кода

Для генерации нужны Buf CLI и Go-плагины `protoc-gen-go` и
`protoc-gen-go-grpc`. Установите Buf CLI по инструкции для вашей платформы на
[странице установки Buf](https://buf.build/docs/installation/).

Установите Go-плагины:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
```

Добавьте каталог с Go-плагинами в `PATH` (для текущего терминала):

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Убедитесь, что все инструменты доступны:

```bash
command -v buf
command -v protoc-gen-go
command -v protoc-gen-go-grpc
```

Затем из каталога `lecture-2-1` выполните:

```bash
make generate
```

Команда генерации описана в `Makefile`. После изменения `.proto` сгенерируйте код заново.
