package pyritego

import (
	"time"
)

func Timer(timeout time.Duration, ch chan bool, expected bool) {
	defer func() {
		// 计时器结束，但 channel 已关闭，阻止 panic 出现
		recover()
	}()

	time.Sleep(timeout)
	ch <- expected
}
