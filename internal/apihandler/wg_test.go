package apihandler

import (
	"sync"
	"testing"
)

func TestWg(t *testing.T)  {
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Done()
	wg.Wait()
}
