# sshclient
SSH Client that allows to customize ciphers on client side.<br>
It supports configuration - see config.txt where you can specify host,user,password.<br>

Usage:<br>
go run main.go -host=host1 => that will pick up credentials from config.txt<br>
go run main.go -host=host1 -user=user -pwd=pass<br>
