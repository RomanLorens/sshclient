package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

type fileConfig struct {
	Host  string   `json:"host"`
	User  string   `json:"user"`
	Pass  string   `json:"pass"`
	Alias []string `json:"alias"`
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
		file := flag.String("file", "C:\\Users\\rl78794\\config.json", "file with hosts and passwords")
		*host, user, password = getCredsFromFile(*host, *file)
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

func getCredsFromFile(hostOrAlias string, file string) (host, user, password string) {
	fmt.Printf("Using %v to resolve credentials for %v host\n", file, hostOrAlias)
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var fc []fileConfig
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(bytes, &fc)
	for _, c := range fc {
		if strings.Contains(c.Host, hostOrAlias) || contains(hostOrAlias, c.Alias) {
			return c.Host, c.User, c.Pass
		}
	}
	panic(fmt.Sprintf("Could not find host configuration for %v host", hostOrAlias))
}

func contains(search string, arr []string) bool {
	for _, l := range arr {
		if l == search {
			return true
		}
	}
	return false
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
