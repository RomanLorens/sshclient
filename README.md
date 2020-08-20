# sshclient
SSH Client that allows to customize ciphers on client side.<br>
It supports configuration - see config.json where you can specify host/alias for host,user,password.<br>

Usage:<br>
go run main.go -host=host1 => that will pick up credentials from config.json<br>
go run main.go -host=alias<br>
go run main.go -host=host1 -user=user -pwd=pass<br>
