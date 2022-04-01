# ATM PoC

This repository contains a sample HTTP REST API for managing an ATM.

This is NOT intended for production use, and is only a test.

## Build

To build the project, you need a go toolchain, and (optionally) GNU make.

```sh
$ make
# will build the server as bin/server
$ ./bin/server
# listens on 0.0.0.0:8080
```

You will also need sqlite3 to build the database and add some rows to make the API usable.
The following code should create the DB, and a user to play with:

```sh
$ ./db_create.sh && sqlite3 db <<EOF
INSERT INTO users(pin, balance) VALUES('4623', 0)
EOF
```

## Test

The service can be tested locally through curl for example, 4 routes are available:

* /login: requires your PIN as a header; ex: `curl -H'nip: 4623' localhost:8080/login`
* /balance: outputs the balance, requires to be authenticated; ex: `curl -H'Authorization: <session-id> localhost:8080/balance'`

* /deposit | /withdrawal: deposits/withdraws funds, POST only, with an amount as body; ex: `curl -d'120' -H'Authorization: <session-id>' localhost:8080/deposit`

NOTE: deposit/withdrawal fail to complete due to a locking issue for now.
