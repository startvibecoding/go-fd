package finder

import "runtime"

func numCPU() int { return runtime.NumCPU() }
