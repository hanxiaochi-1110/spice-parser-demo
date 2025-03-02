package parser

import (
	"bufio"
	"fmt"
	"main.go/models"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Parser struct {
	Netlist   *models.Netlist
	Errors    []models.ParseError
	errorChan chan models.ParseError
	wg        sync.WaitGroup
	mu        sync.Mutex
}

func NewParser() *Parser {
	return &Parser{
		Netlist: &models.Netlist{
			Models: make(map[string]*models.Model),
		},
		errorChan: make(chan models.ParseError, 100),
	}
}

func (p *Parser) ParseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lines := make(chan string, 100)

	// 启动错误收集器
	go func() {
		for err := range p.errorChan {
			p.mu.Lock()
			p.Errors = append(p.Errors, err)
			p.mu.Unlock()
		}
	}()

	// 启动工作池
	numWorkers := runtime.NumCPU()
	p.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go p.worker(lines)
	}

	// 读取文件
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		lines <- fmt.Sprintf("%d|%s", lineNum, scanner.Text())
	}
	close(lines)
	p.wg.Wait()
	close(p.errorChan)

	// 语义检查
	p.semanticCheck()

	return scanner.Err()
}

func (p *Parser) worker(lines <-chan string) {
	defer p.wg.Done()

	for line := range lines {
		parts := strings.SplitN(line, "|", 2)
		lineNum, _ := strconv.Atoi(parts[0])
		content := strings.TrimSpace(parts[1])

		if content == "" || strings.HasPrefix(content, "*") {
			if strings.HasPrefix(content, "*") && p.Netlist.Title == "" {
				p.Netlist.Title = strings.TrimPrefix(content, "* ")
			}
			continue
		}

		if strings.HasPrefix(content, ".model") {
			model, err := parseModel(content)
			if err != nil {
				p.errorChan <- models.ParseError{LineNum: lineNum, Message: err.Error(), Severity: "ERROR"}
				continue
			}
			p.mu.Lock()
			p.Netlist.Models[model.Name] = model
			p.mu.Unlock()
		} else if strings.HasPrefix(content, ".") {
			cmd, err := parseCommand(content)
			if err != nil {
				p.errorChan <- models.ParseError{LineNum: lineNum, Message: err.Error(), Severity: "ERROR"}
				continue
			}
			p.mu.Lock()
			p.Netlist.Commands = append(p.Netlist.Commands, cmd)
			p.mu.Unlock()
		} else {
			comp, err := parseComponent(content)
			if err != nil {
				p.errorChan <- models.ParseError{LineNum: lineNum, Message: err.Error(), Severity: "ERROR"}
				continue
			}
			comp.LineNum = lineNum
			p.mu.Lock()
			p.Netlist.Components = append(p.Netlist.Components, comp)
			p.mu.Unlock()
		}
	}
}

func parseComponent(line string) (*models.Component, error) {
	re := regexp.MustCompile(`(?i)^([A-Z]\w*)\s+(\S+)\s+(\S+)(?:\s+(\S+))?(?:\s+(\S+))?(?:\s+(.*))?$`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("invalid component syntax")
	}

	compType := strings.ToUpper(matches[1][0:1])
	comp := &models.Component{
		Type:   compType,
		Params: make(map[string]float64),
	}

	switch compType {
	case "R", "C", "L":
		comp.Nodes = []string{matches[2], matches[3]}
		if val, err := parseValue(matches[4]); err == nil {
			comp.Params["value"] = val
		} else {
			return nil, err
		}
	case "M":
		comp.Nodes = []string{matches[2], matches[3], matches[4], matches[5]}
		comp.Model = matches[6]
	default:
		return nil, fmt.Errorf("unsupported component type: %s", compType)
	}

	return comp, nil
}

func parseCommand(line string) (*models.Command, error) {
	line = strings.TrimPrefix(line, ".")
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := &models.Command{
		Type:    strings.ToUpper(parts[0]),
		Options: make(map[string]string),
	}

	switch cmd.Type {
	case "TRAN", "AC", "DC":
		for _, opt := range parts[1:] {
			if strings.Contains(opt, "=") {
				kv := strings.SplitN(opt, "=", 2)
				cmd.Options[kv[0]] = kv[1]
			}
		}
	}

	return cmd, nil
}

func parseModel(line string) (*models.Model, error) {
	re := regexp.MustCompile(`(?i)\.model\s+(\w+)\s+(\w+)\s*$(.*)$`)
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("invalid model syntax")
	}

	model := &models.Model{
		Name:   matches[1],
		Type:   matches[2],
		Params: make(map[string]float64),
	}

	params := strings.Fields(matches[3])
	for _, param := range params {
		if strings.Contains(param, "=") {
			kv := strings.Split(param, "=")
			if val, err := parseValue(kv[1]); err == nil {
				model.Params[kv[0]] = val
			}
		}
	}

	return model, nil
}

// 辅助函数
func parseValue(s string) (float64, error) {
	unitMap := map[byte]float64{
		'T': 1e12, 'G': 1e9, 'M': 1e6, 'K': 1e3,
		'm': 1e-3, 'u': 1e-6, 'n': 1e-9, 'p': 1e-12,
	}

	var numPart strings.Builder
	var unitChar byte
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' || c == '-' {
			numPart.WriteByte(byte(c))
		} else {
			unitChar = byte(c)
			break
		}
	}

	val, err := strconv.ParseFloat(numPart.String(), 64)
	if err != nil {
		return 0, err
	}

	if unitChar != 0 {
		multiplier, exists := unitMap[unitChar]
		if !exists {
			return 0, fmt.Errorf("unknown unit: %c", unitChar)
		}
		val *= multiplier
	}

	return val, nil
}

// 语义检查
func (p *Parser) semanticCheck() {
	// 检查模型引用
	for _, comp := range p.Netlist.Components {
		if comp.Model != "" {
			if _, exists := p.Netlist.Models[comp.Model]; !exists {
				p.errorChan <- models.ParseError{
					LineNum:  comp.LineNum,
					Message:  fmt.Sprintf("undefined model: %s", comp.Model),
					Severity: "ERROR",
				}
			}
		}
	}
}
