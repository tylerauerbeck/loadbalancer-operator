package srv

import (
	"context"
)

// runner is a struct that manages the flow of messages for a given loadbalancer
// separate reader and writer channels are utilized in conjunction with a buffer
// to ensure that messages are processed as they are received without blocking
// while the buffer is utilized in order to ensure that only a single message for
// a given loadbalancer is processed at one time
type runner struct {
	reader     chan *lbTask
	writer     chan *lbTask
	quit       chan struct{}
	buffer     []*lbTask
	taskRunner func(*lbTask)
}

type lbTask struct {
	lb  *loadBalancer
	ctx context.Context
	evt string
	srv *Server
}

type taskRunner func(*lbTask)

func (r *runner) stop() {
	// TODO: announce that we're closing down the runner
	r.quit <- struct{}{}
}

// run is executed for each loadbalancer that an operator is responsible for.
// if a message is received on the quit channel, the runner will stop
// if a message is received on the writer channel it is added to the buffer
// if a message is received on the reader channel, a message is removed from the buffer and passed to the taskRunner function
func (r *runner) run() {
	go r.listen()

	for {
		select {
		case <-r.quit:
			return
		default:
			if len(r.buffer) > 0 {
				select {
				case r.reader <- r.buffer[0]:
					r.buffer = r.buffer[1:]
				case d := <-r.writer:
					r.buffer = append(r.buffer, d)
				}
			} else {
				d := <-r.writer
				r.buffer = append(r.buffer, d)
			}
		}
	}
}

// listen pulls messages from the buffer and passes them to the taskRunner function
func (r *runner) listen() {
	for d := range r.reader {
		r.taskRunner(d)
	}
}

func NewRunner(ctx context.Context, tr taskRunner) *runner {
	r := &runner{
		reader:     make(chan *lbTask),
		writer:     make(chan *lbTask),
		buffer:     make([]*lbTask, 0),
		quit:       make(chan struct{}),
		taskRunner: tr,
	}

	go r.run()

	return r
}
