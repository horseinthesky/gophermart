# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Код тестов
https://github.com/Yandex-Practicum/go-autotests/tree/main/cmd/gophermarttest

В корне репы собираем бинарь для тестов:
```
make
```

Далее собираем свой пакет, например, так:
```
go build ./cmd/gophermart/
```

Запускаем тест командой из Github actions вида:
```
~/go-autotests/bin/gophermarttest \
  -test.v -test.run=^TestGophermart$ \
  -gophermart-binary-path=./cmd/gophermart/gophermart \
  -gophermart-host=localhost \
  -gophermart-port=8080 \
  -gophermart-database-uri="postgresql://postgres@localhost:5432?sslmode=disable" \
  -accrual-binary-path=./cmd/accrual/accrual_linux_amd64 \
  -accrual-host=localhost \
  -accrual-port=8000 \
  -accrual-database-uri="postgresql://postgres@localhost:5432?sslmode=disable"
```
