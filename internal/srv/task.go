package srv

func (r *runner) stop() {
	// TODO: announce that we're closing down the runner
	r.quit <- struct{}{}
}

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

func (r *runner) listen() {
	for d := range r.reader {
		r.taskRunner(d)
	}
}

func NewRunner(tr taskRunner) *runner {
	r := &runner{
		reader:     make(chan lbTask),
		writer:     make(chan lbTask),
		buffer:     make([]lbTask, 0),
		quit:       make(chan struct{}),
		taskRunner: tr,
	}

	go r.run()

	return r
}
