package iglocparser

import (
	"errors"
	"sync"
)

var ErrClientsExceed = errors.New("clients exceed")

type EffortParseFn func(client *EffortParserClient, task *EffortParserTask) bool

type EffortParserClient struct {
	*Client
	isInvalidated bool
}

func (self *EffortParserClient) Invalidate() {
	self.isInvalidated = true
}

type EffortParser struct {
	mu sync.Mutex

	clients     chan *EffortParserClient
	clientsLeft int

	tasks     chan *EffortParserTask
	tasksLeft int

	done chan struct{}

	executor EffortParseFn
}

func (self *EffortParser) isClientsExceed() bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.clientsLeft <= 0
}

func (self *EffortParser) decreaseClients() int {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.clientsLeft--
	return self.clientsLeft
}

func (self *EffortParser) isTasksExceed() bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.tasksLeft <= 0
}

func (self *EffortParser) decreaseTasks() int {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.tasksLeft--
	return self.tasksLeft
}

func (self *EffortParser) Run() error {
	for {
		select {
		case <-self.done:
			return nil
		case task := <-self.tasks:
			if self.isClientsExceed() {
				return ErrClientsExceed
			}

			go self.executeTask(<-self.clients, task)
		}
	}
}

func (self *EffortParser) executeTask(client *EffortParserClient, task *EffortParserTask) {
	if !task.IsCanUseClient(client) {
		self.clients <- client
		return
	}

	isDone := self.executor(client, task)
	if client.isInvalidated {
		self.decreaseClients()
	} else {
		self.clients <- client
	}

	if isDone {
		self.tryToDone()
	} else {
		self.tasks <- task
	}
}

func (self *EffortParser) tryToDone() {
	if self.decreaseTasks() <= 0 {
		close(self.done)
	}
}

func NewEffortParser(clients []*Client, tasks []interface{}, attempts int, fn EffortParseFn) *EffortParser {
	parser := &EffortParser{
		clients:     make(chan *EffortParserClient, len(clients)),
		clientsLeft: len(clients),

		tasks:     make(chan *EffortParserTask, len(tasks)),
		tasksLeft: len(tasks),

		done: make(chan struct{}),

		executor: fn,
	}

	for _, client := range clients {
		parser.clients <- &EffortParserClient{Client: client}
	}

	for _, task := range tasks {
		parser.tasks <- &EffortParserTask{
			Data: task,

			attemptsLeft: attempts,

			parser: parser,
		}
	}

	return parser
}

type EffortParserTask struct {
	Data interface{}

	mu             sync.Mutex
	attemptsLeft   int
	utilizeClients map[*EffortParserClient]struct{}

	parser *EffortParser

	isDone bool
}

func NewEffortParserTask(data interface{}, attempts int) *EffortParserTask {
	return &EffortParserTask{
		Data: data,

		attemptsLeft: attempts,
	}
}

func (self *EffortParserTask) AttemptsLeft() int {
	return self.attemptsLeft
}

func (self *EffortParserTask) AttemptsDecrease() int {
	self.attemptsLeft--
	return self.attemptsLeft
}

func (self *EffortParserTask) IsValid() bool {
	return self.attemptsLeft <= 0 || len(self.utilizeClients) >= cap(self.parser.clients)
}

func (self *EffortParserTask) DontUseClient(client *EffortParserClient) {
	if self.utilizeClients == nil {
		self.utilizeClients = make(map[*EffortParserClient]struct{})
	}

	self.utilizeClients[client] = struct{}{}
}

func (self *EffortParserTask) IsCanUseClient(client *EffortParserClient) bool {
	if self.utilizeClients == nil {
		return true
	}

	_, ok := self.utilizeClients[client]
	return !ok
}
