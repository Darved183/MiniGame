package game

import (
	"MyGame/sound"
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

const (
	MsgInit   = "INIT"
	MsgState  = "STATE"
	MsgAction = "ACTION"
	MsgChat   = "CHAT"
	MsgEnd    = "END"
)

type Init struct {
	P1Name string
	P1HP   int
	P1Max  int
	P2Name string
	P2HP   int
	P2Max  int
	Round  int
	Turn   int
}

type State struct {
	Round int
	P1HP  int
	P2HP  int
	Turn  int
}

type Action struct {
	Kind      string
	BodyPart  string
	BlockPart string
	Damage    int
	ItemIdx   int
}

type End struct {
	Winner int
}

type Session struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
}

func ParseInit(line string) (Init, error) {
	var i Init
	i.P1Name, i.P2Name, i.P1Max, i.P2Max = "Игрок1", "Игрок2", 100, 100
	for _, part := range strings.Fields(line) {
		if strings.HasPrefix(part, "p1name=") {
			i.P1Name = part[7:]
		} else if strings.HasPrefix(part, "p2name=") {
			i.P2Name = part[7:]
		} else if strings.HasPrefix(part, "p1hp=") {
			i.P1HP, _ = strconv.Atoi(part[5:])
		} else if strings.HasPrefix(part, "p1max=") {
			i.P1Max, _ = strconv.Atoi(part[6:])
		} else if strings.HasPrefix(part, "p2hp=") {
			i.P2HP, _ = strconv.Atoi(part[5:])
		} else if strings.HasPrefix(part, "p2max=") {
			i.P2Max, _ = strconv.Atoi(part[6:])
		} else if strings.HasPrefix(part, "round=") {
			i.Round, _ = strconv.Atoi(part[6:])
		} else if strings.HasPrefix(part, "turn=") {
			i.Turn, _ = strconv.Atoi(part[5:])
		}
	}
	return i, nil
}

func SerializeInit(i Init) string {
	return fmt.Sprintf("INIT p1name=%s p1hp=%d p1max=%d p2name=%s p2hp=%d p2max=%d round=%d turn=%d",
		i.P1Name, i.P1HP, i.P1Max, i.P2Name, i.P2HP, i.P2Max, i.Round, i.Turn)
}

func ParseState(line string) (State, error) {
	var s State
	for _, part := range strings.Fields(line) {
		if strings.HasPrefix(part, "round=") {
			s.Round, _ = strconv.Atoi(part[6:])
		} else if strings.HasPrefix(part, "p1hp=") {
			s.P1HP, _ = strconv.Atoi(part[5:])
		} else if strings.HasPrefix(part, "p2hp=") {
			s.P2HP, _ = strconv.Atoi(part[5:])
		} else if strings.HasPrefix(part, "turn=") {
			s.Turn, _ = strconv.Atoi(part[5:])
		}
	}
	return s, nil
}

func SerializeState(s State) string {
	return fmt.Sprintf("STATE round=%d p1hp=%d p2hp=%d turn=%d", s.Round, s.P1HP, s.P2HP, s.Turn)
}

func ParseAction(line string) (Action, error) {
	parts := strings.SplitN(line, " ", 3)
	var a Action
	if len(parts) < 2 {
		return a, fmt.Errorf("invalid action")
	}
	a.Kind = parts[1]
	if a.Kind == "attack" && len(parts) >= 3 {
		sub := strings.Split(parts[2], "|")
		if len(sub) > 0 {
			a.BodyPart = sub[0]
		}
		if len(sub) > 1 {
			a.BlockPart = sub[1]
		}
		if len(sub) > 2 {
			a.Damage, _ = strconv.Atoi(sub[2])
		}
	} else if a.Kind == "item" && len(parts) >= 3 {
		a.ItemIdx, _ = strconv.Atoi(parts[2])
	}
	return a, nil
}

func SerializeAction(a Action) string {
	switch a.Kind {
	case "attack":
		return fmt.Sprintf("ACTION attack %s|%s|%d", a.BodyPart, a.BlockPart, a.Damage)
	case "item":
		return fmt.Sprintf("ACTION item %d", a.ItemIdx)
	case "surrender":
		return "ACTION surrender"
	}
	return ""
}

func ParseEnd(line string) (End, error) {
	var e End
	for _, part := range strings.Fields(line) {
		if strings.HasPrefix(part, "winner=") {
			e.Winner, _ = strconv.Atoi(part[7:])
			break
		}
	}
	return e, nil
}

func SerializeEnd(e End) string {
	return fmt.Sprintf("END winner=%d", e.Winner)
}

func NewSession(conn net.Conn) *Session {
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
	}
	return &Session{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, 4096),
	}
}

func (s *Session) WriteLine(line string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return io.ErrClosedPipe
	}
	_, err := io.WriteString(s.conn, line+"\n")
	return err
}

func (s *Session) ReadLine() (string, error) {
	if s.reader == nil {
		return "", io.ErrClosedPipe
	}
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sound.StopMusic()
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	s.reader = nil
	return err
}
