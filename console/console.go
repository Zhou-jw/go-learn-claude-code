package console

import (
	"fmt"

	"github.com/fatih/color"
)

// 全局开关（像 DPrintf）
var Enable = true

func Info(format string, a ...any) {
	if len(a) == 0 {
		fmt.Println(format)
		return
	}
	fmt.Printf(format, a...)
}

// Yellow 黄色打印
func Yellow(format string, a ...any) {
	if Enable {
		color.Yellow(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// Red 红色打印
func Red(format string, a ...any) {
	if Enable {
		color.Red(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// Green 绿色打印
func Green(format string, a ...any) {
	if Enable {
		color.Green(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

// Cyan 青色打印
func Cyan(format string, a ...any) {
	if Enable {
		color.Cyan(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}
