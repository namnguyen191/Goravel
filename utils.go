package goravel

import (
	"fmt"
	"regexp"
	"runtime"
	"time"
)

func (grv *Goravel) LoadTime(start time.Time) {
	elapsed := time.Since(start)
	pc, _, _, _ := runtime.Caller(1)
	funcObj := runtime.FuncForPC(pc)
	runtimeFunc := regexp.MustCompile(`^.*\.(.*)$`)
	name := runtimeFunc.ReplaceAllString(funcObj.Name(), "$1")

	grv.InfoLog.Println(fmt.Sprintf("Load Time: %s took %s", name, elapsed))
}
