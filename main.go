package main

import (
	"fmt"
	"sync"
	"time"
)

//Fibonacci returns the fibonacci of a number, using recursion.
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}

//number to calculate Fibonacci
type Job struct {
	param int
}

//worker is a goroutine that is going to execute the work
type Worker struct {
	id          int
	Dis         *Dispatcher
	QuitChannel chan bool
}

//dispatcher is the config of the workers. it also stores the list of jobs.
type Dispatcher struct {
	MaxWorkers int
	JobQueue   chan Job
	Results    map[int]int
	Pending    map[int]bool
	Wg         *sync.RWMutex
	WaitGroup  *sync.WaitGroup
	IsCacheOn  bool
}

//StartWorker worker constructor
func StartWorker(id int, dis *Dispatcher) *Worker {
	return &Worker{
		id:          id,
		Dis:         dis,
		QuitChannel: make(chan bool),
	}
}

//StartDispatcher dispatcher constructor
func StartDispatcher(MaxWorkers int, iso bool) *Dispatcher {
	var wg sync.RWMutex
	var WaitGroup sync.WaitGroup
	return &Dispatcher{
		MaxWorkers: MaxWorkers,
		JobQueue:   make(chan Job, 20),
		Results:    make(map[int]int),
		Pending:    make(map[int]bool),
		Wg:         &wg,
		IsCacheOn:  iso,
		WaitGroup:  &WaitGroup,
	}
}

//GetFibo prints the fibonacci of a number
func (w *Worker) GetFibo(n int) {
	w.Dis.Wg.Lock()
	w.Dis.Pending[n] = true
	w.Dis.Wg.Unlock()
	fiboRes := Fibonacci(n)
	w.Dis.Wg.Lock()
	w.Dis.Results[n] = fiboRes
	w.Dis.Pending[n] = false
	w.Dis.Wg.Unlock()
	fmt.Printf("From worker with ID %v, the result of %v is %v\n", w.id, n, fiboRes)
}

//Work is listening for a new Job. If the job was already processed or is being processed, print it. If not, execute GetFibo()
func (w *Worker) Work() {
	go func() {
		for {
			select {
			case job := <-w.Dis.JobQueue:
				if w.Dis.IsCacheOn {
					w.Dis.Wg.RLock()
					res, exist := w.Dis.Results[job.param]
					w.Dis.Wg.RUnlock()
					if exist {
						fmt.Printf("From worker with ID %v, the result of %v is %v (getting info from cache)\n", w.id, job.param, res)
					} else {
						w.Dis.Wg.RLock()
						res := w.Dis.Pending[job.param]
						w.Dis.Wg.RUnlock()
						if res {
							fmt.Printf("From worker with ID %v, the result of %v is being processed\n", w.id, job.param)
						} else {
							w.GetFibo(job.param)
						}
					}
				} else {
					w.GetFibo(job.param)
				}

				w.Dis.WaitGroup.Done()
			case <-w.QuitChannel:
				fmt.Printf("Worker with ID %v stopped\n", w.id)
				return
			}
		}
	}()
}

//StopWorker stops 1 worker
func (w *Worker) StopWorker() {
	go func() {
		w.QuitChannel <- true
	}()
}

func main() {
	isCacheOn := false // set to false if you want to see the difference with a no-cache system.
	fibo := []int{42, 40, 41, 42, 38, 9, 11, 23, 25, 42, 42, 11, 42, 42, 43, 43, 38}
	currentDis := StartDispatcher(5, isCacheOn)
	start := time.Now()
	for i := 0; i < currentDis.MaxWorkers; i++ {
		go func(index int) {
			worker := StartWorker(index+1, currentDis)
			worker.Work()
			if index == 2 {
				worker.StopWorker()
			}
		}(i)
	}

	for _, n := range fibo {
		currentDis.WaitGroup.Add(1)
		job := Job{n}
		currentDis.JobQueue <- job
	}
	currentDis.WaitGroup.Wait()
	fmt.Println("Total time:", time.Since(start))

}
