package hystrix

import "testing"
import "time"
import "errors"

func TestExecute(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Result: 1} },
		func(err error, result_channel chan Result) { result_channel <- Result{Error: nil} },
	)
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

func TestQueue(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Result: 1} },
		func(err error, result_channel chan Result) { result_channel <- Result{Error: nil} },
	)
	future := command.Queue()
	if future.Value().Result != 1 {
		t.Fail()
	}
}

func TestObserve(t *testing.T) {
	run_func := func(results chan Result) {
		results <- Result{Result: 1}
		results <- Result{Result: 2}
		results <- Result{Result: 3}
		close(results)
	}

	validation := make(chan int, 3)
	observer_func := func(result Result) {
		validation <- result.Result.(int)
	}

	command := NewCommand(run_func, nil)
	command.Observer = observer_func
	command.Observe()

	time.Sleep(10 * time.Millisecond)

	sum := <-validation + <-validation + <-validation
	if sum != 6 {
		t.Fail()
	}
}

func TestFallbackMissing(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Error: errors.New("failure")} },
		nil,
	)
	result := command.Execute()
	if !(result.Result == nil && result.Error.Error() == "failure") {
		t.Fail()
	}
}

func TestFallback(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Error: errors.New("sup")} },
		func(err error, result_channel chan Result) { result_channel <- Result{Result: 1} },
	)
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

// TODO: how can we be sure the fallback is triggered from timeout.  error type?
func TestTimeout(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { time.Sleep(1 * time.Second); result_channel <- Result{Result: 2} },
		func(err error, result_channel chan Result) { result_channel <- Result{Result: 1} },
	)
	result := command.Execute()
	if result.Result != 1 {
		t.Fail()
	}
}

// TODO: how can we be sure the fallback is triggered from full pool.  error type?
func TestFullExecutorPool(t *testing.T) {
	pool := NewExecutorPool("TestFullExecutorPool", 2)

	command1 := NewCommand(
		func(result_channel chan Result) { time.Sleep(10 * time.Millisecond) },
		func(err error, result_channel chan Result) { result_channel <- Result{Result: 1} },
	)
	command1.ExecutorPool = pool
	command2 := NewCommand(
		func(result_channel chan Result) { time.Sleep(10 * time.Millisecond) },
		func(err error, result_channel chan Result) { result_channel <- Result{Result: 1} },
	)
	command2.ExecutorPool = pool
	command3 := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Result: 2} },
		func(err error, result_channel chan Result) { result_channel <- Result{Result: 1} },
	)
	command3.ExecutorPool = pool

	command1.Queue()
	command2.Queue()
	result := command3.Execute()

	if result.Result != 1 {
		t.Fail()
	}
}

func TestOpenCircuit(t *testing.T) {
	command := NewCommand(
		func(result_channel chan Result) { result_channel <- Result{Result: 1} },
		nil,
	)
	command.ExecutorPool.Circuit.IsOpen = true
	result := command.Execute()
	if result.Error == nil {
		t.Fail()
	}

	// BUG: the executor pool is not naturally reset between tests
	executor_pools = make(map[string]*ExecutorPool)
}
