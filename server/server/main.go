package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	chatPort   = 8081
	pvpPort    = 7000
	chatBufLen = 10
)

var (
	chatClients = make(map[net.Conn]bool)
	chatMu      sync.RWMutex
	chatMsgChan = make(chan string, chatBufLen)
)

func portFromEnv(name string, defaultPort int) int {
	p := os.Getenv(name)
	if p == "" {
		return defaultPort
	}
	v, err := strconv.Atoi(p)
	if err != nil || v <= 0 {
		return defaultPort
	}
	return v
}

func main() {
	chatPort := portFromEnv("CHAT_PORT", chatPort)
	pvpPort := portFromEnv("PVP_PORT", pvpPort)

	go runChatServer(chatPort)
	runPvPServer(pvpPort)
}

func runChatServer(port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Чат: не удалось слушать :%d: %v\n", port, err)
		return
	}
	defer listener.Close()
	fmt.Printf("Чат запущен на :%d\n", port)

	go chatBroadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go chatHandleClient(conn)
	}
}

func chatHandleClient(conn net.Conn) {
	defer func() {
		chatMu.Lock()
		delete(chatClients, conn)
		chatMu.Unlock()
		conn.Close()
	}()

	chatMu.Lock()
	chatClients[conn] = true
	chatMu.Unlock()

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		chatMsgChan <- fmt.Sprintf("[%s] %s", conn.RemoteAddr(), msg)
	}
}

func chatBroadcaster() {
	for msg := range chatMsgChan {
		chatMu.RLock()
		for conn := range chatClients {
			_, _ = conn.Write([]byte(msg))
		}
		chatMu.RUnlock()
		fmt.Print(msg)
	}
}

func runPvPServer(port int) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PvP: ошибка запуска на %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer ln.Close()
	fmt.Printf("PvP запущен на %s. Ожидание двух игроков...\n", addr)

	for {
		conn1, err := ln.Accept()
		if err != nil {
			continue
		}
		fmt.Printf("[PvP 1] Подключён: %s\n", conn1.RemoteAddr())

		conn2, err := ln.Accept()
		if err != nil {
			conn1.Close()
			continue
		}
		fmt.Printf("[PvP 2] Подключён: %s. Начинаем бой.\n", conn2.RemoteAddr())
		go runPvPMatch(conn1, conn2)
	}
}

func runPvPMatch(conn1, conn2 net.Conn) {
	defer conn1.Close()
	defer conn2.Close()

	initLine := "INIT p1name=Игрок1 p1hp=100 p1max=100 p2name=Игрок2 p2hp=100 p2max=100 round=1 turn=1"

	if _, err := io.WriteString(conn1, "YOU_ARE 1\n"); err != nil {
		return
	}
	if _, err := io.WriteString(conn1, initLine+"\n"); err != nil {
		return
	}

	if _, err := io.WriteString(conn2, "YOU_ARE 2\n"); err != nil {
		return
	}
	if _, err := io.WriteString(conn2, initLine+"\n"); err != nil {
		return
	}

	var wg sync.WaitGroup
	relay := func(from, to net.Conn) {
		defer wg.Done()
		rd := bufio.NewReaderSize(from, 4096)
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			if _, err := io.WriteString(to, line+"\n"); err != nil {
				return
			}
			fmt.Printf("[PvP relay %s -> %s] %s\n", from.RemoteAddr(), to.RemoteAddr(), line)
		}
	}
	wg.Add(2)
	go relay(conn1, conn2)
	go relay(conn2, conn1)
	wg.Wait()
}
