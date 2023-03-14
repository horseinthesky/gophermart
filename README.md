# 🔄 gophermart

Cumulative loyalty system.

## ✨ Features

- 🔒 Register and authenticate with [JWT](https://jwt.io/) or [PASETO](https://paseto.io/) token
- 💻 Add new orders
- 📚 Maintain a list of user's orders
- 📋 Maintain user loyalty account balance
- 🔌 Verify accepted order numbers through the loyalty points system
- 📊 Get accrual of the required reward for each matching order number to the user's loyalty account

Check out the `SPECIFICATION.md` file for business logic details.

## 📊 AutoTests

Project autotests are available here:
https://github.com/Yandex-Practicum/go-autotests/tree/main/cmd/gophermarttest

### Updates

To be able to get updates for the test suite run:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

To update the test suite source code run:

```
git fetch template && git checkout template/master .github
```

Then add changes to your repo.

### Run

To build a test suite binary run:

```
make
```

Next build gophermart with:

```
go build ./cmd/gophermart/
```

Run tests with the following command:

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
