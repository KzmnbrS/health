package health

import "time"

type Check interface {
	Do(v interface{})
	Interval() time.Duration
}

var checks []Check

// AddCheck adds the new check to run. Check func will receive operational state pointer.
func AddCheck(ch Check) {
	mtx.Lock()
	checks = append(checks, ch)
	mtx.Unlock()
}

func startChecks() {
	for _, ch := range checks {
		go func(ch Check) {
			for {
				if GetStatus() != StatusUp {
					return
				}

				ch.Do(opState)
				<-time.After(ch.Interval())
			}
		}(ch)
	}
}
