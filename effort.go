package iglocparser

import (
	"sync"
)

type EffortParseFn func(client *Client, task *EffortParserTask)

type EffortParser struct {
	clients chan *Client
	tasks   chan *EffortParserTask
	done    chan struct{}

	fn EffortParseFn

	mu   sync.Mutex
	left int
}

func NewEffortParser(clients []*Client, tasks []interface{}, attempts int, fn EffortParseFn) *EffortParser {
	parser := &EffortParser{
		clients: make(chan *Client, len(clients)),
		tasks:   make(chan *EffortParserTask, len(tasks)),
		done:    make(chan struct{}),

		fn: fn,

		left: len(tasks),
	}

	for _, client := range clients {
		parser.clients <- client
	}

	for _, task := range tasks {
		parser.tasks <- &EffortParserTask{
			Data:     task,
			attempts: attempts,
			parser:   parser,
		}
	}

	return parser
}

type EffortParserTask struct {
	Data     interface{}
	attempts int

	mu             sync.Mutex
	utilizeClients map[*Client]struct{}

	parser *EffortParser

	isDone bool
}

func NewEffortParserTask(data interface{}, attempts int) *EffortParserTask {
	return &EffortParserTask{
		Data:     data,
		attempts: attempts,
	}
}

func (self *EffortParserTask) Attempts() int {
	return self.attempts
}

func (self *EffortParserTask) Exceed() bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.attempts <= 0 || len(self.utilizeClients) >= cap(self.parser.clients)
}

// помечает задачку как выполненную и удаляет из пула
func (self *EffortParserTask) Done() {
	if self.isDone {
		return
	}

	self.isDone = true
	self.parser.tryToDone()
}

// проверяет если остались попытки и использованы ещё не все клиенты
// то помечает задачу как невыполненную и возвращает в пул
// результат работы обозначает вернулась ли задача в пул или была удалена из пула
func (self *EffortParserTask) Undone() bool {
	if self.isDone {
		return false
	}

	if self.Exceed() {
		self.isDone = true
		self.parser.tryToDone()

		return false
	}

	self.parser.tasks <- self
	return true
}

func (self *EffortParserTask) Utilize(client *Client) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.utilizeClients == nil {
		self.utilizeClients = make(map[*Client]struct{})
	}

	self.utilizeClients[client] = struct{}{}
}

func (self *EffortParserTask) isUtilize(client *Client) bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	if self.utilizeClients == nil {
		return false
	}

	_, ok := self.utilizeClients[client]
	return ok
}

func (self *EffortParser) Run() {
	for {
		select {
		case <-self.done:
			return
		case task := <-self.tasks:
			go self.do(<-self.clients, task)
		}
	}
}

func (self *EffortParser) do(client *Client, task *EffortParserTask) {
	defer func() { self.clients <- client }()
	if task.isUtilize(client) {
		return
	}

	self.fn(client, task)

	if !task.isDone {
		task.Utilize(client)
	}
}

func (self *EffortParser) tryToDone() {
	self.left--
	if self.left <= 0 {
		close(self.done)
	}
}
