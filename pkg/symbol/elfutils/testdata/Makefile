all:
	GOOS=linux GOARCH=amd64 go build -ldflags "-w -s" main.go
	strip -R ".gosymtab" main
