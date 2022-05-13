package health

import "time"

type check struct {
	fn       func(opState interface{})
	interval time.Duration
}

var checks []check

// AddCheck adds the new check to run at the given interval.
// Check func will receive operational state pointer.
func AddCheck(fn func(opState interface{}), interval time.Duration) {
	mtx.Lock()
	checks = append(checks, check{fn, interval})
	mtx.Unlock()
}

func startChecks() {
	for _, ch := range checks {
		go func(ch check) {
			for {
				if GetStatus() != StatusUp {
					return
				}

				ch.fn(opState)
				<-time.After(ch.interval)
			}
		}(ch)
	}
}
