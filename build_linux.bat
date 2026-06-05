REM Requires x86_64-linux-gnu-gcc in PATH because go-sqlite3 uses CGO.
set CGO_ENABLED=1
set CC=x86_64-linux-gnu-gcc
set GOOS=linux
set GOARCH=amd64

go build -o doc main.go

set CGO_ENABLED=1
set CC=
set GOOS=windows
set GOARCH=amd64
