package system

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
)

type system struct {
	fn       []any
	mainChan chan error
	sigint   chan os.Signal
	sync.RWMutex
}

type System interface {
	Run(func() error) System
	Register(fn ...any) System
	Wait(fn ...any)
	Close()
}

func New() System {
	return &system{
		mainChan: make(chan error),
	}
}

func (c *system) Register(fn ...any) System {
	c.Lock()
	defer c.Unlock()
	c.fn = append(c.fn, fn...)
	return c
}

func (c *system) Close() {
	c.Lock()
	defer c.Unlock()
	if c.sigint == nil {
		return
	}
	c.sigint <- os.Interrupt
}

func (c *system) Run(mainFn func() error) System {
	go func() {
		err := mainFn()
		if err != nil {
			c.mainChan <- err
		}
	}()
	return c
}

func (c *system) closeAll() {
	for _, input := range c.fn {
		if val, ok := input.(func() error); ok {
			err := val()
			if err != nil {
				log.Printf("Error while closing: %s\n", err.Error())
			}
		} else if val, ok := input.(func(ctx context.Context) error); ok {
			err := val(context.Background())
			if err != nil {
				log.Printf("Error while closing: %s\n", err.Error())
			}
		} else if val, ok := input.(func()); ok {
			val()
		} else if val, ok := input.(func(ctx context.Context)); ok {
			val(context.Background())
		}
	}
}

func (c *system) Wait(fn ...any) {
	if len(fn) > 0 {
		c.Register(fn...)
	}

	idleConnClosed := make(chan struct{})
	go func() {
		c.Lock()
		c.sigint = make(chan os.Signal, 1)
		signal.Notify(c.sigint, os.Interrupt, os.Kill)
		c.Unlock()

		select {
		case err := <-c.mainChan:
			fmt.Print("\r")
			log.Printf("Received Error: %s\n", err.Error())
			c.closeAll()
		case ss := <-c.sigint:
			fmt.Print("\r")
			log.Printf("Received Signal %s\n", ss.String())
			c.closeAll()
		}
		close(idleConnClosed)
	}()

	<-idleConnClosed
	log.Printf("All System Has Been Shutdown\n")
}
