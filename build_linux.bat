set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

go build -o doc main.go

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64