package runtime

import (
	"github.com/995933447/easytask/internal/util/logger"
	"runtime/debug"
	"strings"
	"context"
	"fmt"
	"os"
)

func RecoverToTraceAndExit(ctx context.Context) {
	if err := recover(); err != nil {
		fmt.Println(err)
		logger.MustGetSysLogger().Errorf(ctx, "PROCESS PANIC: err %s", err)
		stack := debug.Stack()
		if len(stack) > 0 {
			lines := strings.Split(string(stack), "\n")
			for _, line := range lines {
				fmt.Println(line)
				logger.MustGetSysLogger().Error(ctx, line)
			}
		}
		logger.MustGetSysLogger().Errorf(ctx, "stack is empty (%s)", err)
		os.Exit(0)
	}
}
