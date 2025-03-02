package main

import (
	"encoding/json"
	"fmt"
	"main.go/parser"
	"os"
)

// 具体解析实现

// 主函数
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: spice-parser <input.sp>")
		os.Exit(1)
	}

	psr := parser.NewParser()
	if err := psr.ParseFile(os.Args[1]); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	jsonData, _ := json.MarshalIndent(psr.Netlist, "", "  ")
	fmt.Println("Parsed Netlist:")
	fmt.Println(string(jsonData))

	// 输出错误
	if len(psr.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range psr.Errors {
			fmt.Printf("[%s] Line %d: %s\n", err.Severity, err.LineNum, err.Message)
		}
	}
}
