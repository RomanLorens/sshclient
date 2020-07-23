# sshclient
SSH Client that allows to customize ciphers on client side.
It supports configuration - see config.txt where you can specify host,user,password.

Usage:
go run main.go -host=host1 => that will pick up credentials from config.txt
go run main.go -host=host1 -user=user -pwd=pass
