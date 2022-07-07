package runner

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"
)

/*
a
	b
	c
		d
		e
*/

type IRunner interface {
	Name() string
	Init() error
	Run() error
	SetName(name string) IRunner
}

type Runner struct {
	ctx      context.Context
	runner   IRunner
	timeout  time.Duration
	children []*Runner
	defers   []func() error
	status   string
	duration time.Duration
}

func New(ctx context.Context, ir IRunner) *Runner {
	r := &Runner{
		ctx:     ctx,
		timeout: time.Duration(10) * time.Second,
	}
	if ir != nil {
		r.runner = ir
	}
	return r
}

func NewWithTimeout(ctx context.Context, ir IRunner, timeout time.Duration) *Runner {
	r := &Runner{
		timeout: timeout,
	}
	if ir != nil {
		r.runner = ir
	}
	return r
}

func (r *Runner) WithTimeout(timeout time.Duration) *Runner {
	r.timeout = timeout
	return r
}

func (r *Runner) Run() (err error) {
	deadline := time.Now().Add(r.timeout)
	runTimeStart := time.Now()
	var runTimeEnd time.Time
	defer func() {
		r.duration = time.Since(runTimeStart)
		rd := int64(0)
		if runTimeEnd.Unix() > 0 {
			rd = runTimeEnd.Sub(runTimeStart).Milliseconds()
		}
		log.Printf(
			"runner end: %s, duration: {run: %dms, total: %dms}\n",
			r.runner.Name(),
			rd,
			time.Since(runTimeStart).Milliseconds(),
		)
	}()
	if r.runner != nil {
		if err = r.runner.Init(); err != nil {
			return
		}
		err = r.runner.Run()
		runTimeEnd = time.Now()
		if err != nil {
			return
		}
	}
	if r.timeout > 0 && time.Now().After(deadline) {
		r.status = "timeout"
		return
	}
	runnerNum := len(r.children)
	if len(r.children) > 0 {
		ch := make(chan int, runnerNum)
		timeStart := time.Now()
		var timeout time.Duration
		for _, runner := range r.children {
			if timeout < runner.timeout {
				timeout = runner.timeout
			}
			go func(runner *Runner) {
				var re error
				defer func() {
					if err := recover(); err != nil {
						stack := make([]byte, 10000)
						runtime.Stack(stack, false)
						ch <- 0
						runner.status = "panic"
					} else {
						if re != nil {
							ch <- 0
							runner.status = "error"
						} else {
							ch <- 1
							runner.status = "ok"
						}
					}
					runner.duration = time.Since(timeStart)
				}()
				re = runner.Run()
			}(runner)
		}
		succNum := 0
		failNum := 0
		tick := time.After(timeout)
		for {
			select {
			case status := <-ch:
				if status > 0 {
					succNum++
				} else {
					failNum++
				}
				if succNum+failNum == runnerNum {
					goto end
				}
			case <-tick:
				goto end
			}
		}
	end:
		//infos := []string{}
		for _, c := range r.children {
			var cs string
			status := c.status
			if status != "ok" {
				if status == "running" {
					status = "timeout"
					c.duration = timeout
				}
				cs = c.String()
				log.Printf("rpc [%s]: %s\n", status, cs)
			} else {
				cs = c.String()
				log.Printf("rpc [%s]: %s\n", status, cs)
			}
		}
		log.Printf("end: total %v, succ:%v, fail:%v, timeout:%v\n", runnerNum, succNum, failNum, runnerNum-succNum-failNum)
	}
	if len(r.defers) > 0 {
		for _, fn := range r.defers {
			fn()
		}
	}
	return
}

func (r *Runner) DepOn(p *Runner) {
	p.children = append(p.children, r)
}

func (r *Runner) Done(fn func() error) {
	r.defers = append(r.defers, fn)
}

func (r *Runner) SetName(name string) *Runner {
	r.runner.SetName(name)
	return r
}

func (r *Runner) String() string {
	status := r.status
	if status == "" {
		status = "running"
	}
	return fmt.Sprintf("{id:%s, status:%s, duration:%dms}", r.runner.Name(), status, r.duration.Milliseconds())
}

type simple struct {
	name string
	fn   func() error
}

func (s *simple) Name() string {
	return s.name
}

func (s *simple) SetName(name string) IRunner {
	s.name = name
	return s
}

func (s *simple) Init() error {
	return nil
}

func (s *simple) Run() error {
	return s.fn()
}

func Wrap(ctx context.Context, fn ...func() error) *Runner {
	_fn := func() error { return nil }
	if len(fn) > 0 {
		_fn = fn[0]
	}
	_, name, line, _ := runtime.Caller(1)
	nl := len(name)
	if nl > 52 {
		name = ".." + name[nl-50:]
	}
	name = fmt.Sprintf("%s:%v", name, line)
	return New(ctx, &simple{
		name: name,
		fn:   _fn,
	})
}
