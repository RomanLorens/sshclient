package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type config struct {
	user     string
	password string
	host     string
	ciphers  []string
}

func main() {
	cfg := getConfig()
	wr, sess := connect(cfg)

	cmds := make(chan string)
	errc := make(chan string)

	go commands(cmds, errc)

	//add alias
	go func() {
		cmds <- "alias ll='ls -lahtr'"
	}()

	for {
		select {
		case cmd := <-cmds:
			_, err := fmt.Fprintf(wr, "%s\n", cmd)
			if err != nil {
				panic(err)
			}
		case err := <-errc:
			fmt.Println("Closing connection...", err)
			sess.Close()
			if e := sess.Wait(); e != nil {
				panic(e)
			}
			return
		}
	}
}

func getConfig() *config {
	host := flag.String("host", "", "hostname:port")
	var ciphers string
	flag.StringVar(&ciphers, "c", "", "client ciphers comma separated")
	var user, password string
	flag.StringVar(&user, "user", "", "user")
	flag.StringVar(&password, "pwd", "", "pwd")
	flag.Parse()

	if *host == "" {
		panic("Required -host option")
	}
	if user == "" || password == "" {
		file := flag.String("file", "config.txt", "file with hosts and passwords")
		user, password = getCredsFromFile(*host, *file)
	}

	if !strings.Contains(":", *host) {
		*host = *host + ":22"
	}
	cfg := config{
		user:     user,
		password: password,
		host:     *host,
	}
	if ciphers != "" {
		cfg.ciphers = make([]string, 0)
		for _, c := range strings.Split(ciphers, ",") {
			cfg.ciphers = append(cfg.ciphers, strings.TrimSpace(c))
		}
	}
	return &cfg
}

func getCredsFromFile(host string, file string) (user, password string) {
	fmt.Printf("Using %v to resolve credentials for %v host\n", file, host)
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		if len(tokens) == 3 && strings.Contains(tokens[0], host) {
			user = tokens[1]
			password = tokens[2]
			break
		}
	}
	if user == "" {
		panic(fmt.Sprintf("Could not find host configuration for %v host", host))
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return user, password
}

func connect(cfg *config) (io.WriteCloser, *ssh.Session) {
	var config ssh.Config
	config.SetDefaults()
	config.Ciphers = append(config.Ciphers, "3des-cbc")
	if len(cfg.ciphers) > 0 {
		config.Ciphers = append(config.Ciphers, cfg.ciphers...)
	}
	fmt.Println("Client ciphers", config.Ciphers)

	sshConfig := &ssh.ClientConfig{
		User: cfg.user,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(password(cfg.password)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Config:          config,
	}
	conn, err := ssh.Dial("tcp", cfg.host, sshConfig)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected...")
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	stdin, err := sess.StdinPipe()
	if err != nil {
		panic(err)
	}

	err = sess.Shell()
	if err != nil {
		panic(err)
	}
	return stdin, sess
}

func password(pwd string) func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	return func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
		answers = make([]string, len(questions))
		for n := range questions {
			answers[n] = pwd
		}
		return answers, nil
	}
}

func commands(out chan string, errc chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(">> ")
	for scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "exit" || cmd == "bye" {
			errc <- "exit"
			return
		}
		out <- scanner.Text()
		fmt.Print(">> ")
	}
	if err := scanner.Err(); err != nil {
		errc <- err.Error()
	}
}
