package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	gs "github.com/shirou/gopsutil/v3/process"
)

func GetProcessList() []table.Row {
	process_list, err := gs.Processes()
	if err != nil {
		fmt.Println("getProcessList error: ", err.Error())
		os.Exit(-1)
	}

	results := []table.Row{}

	for _, process := range process_list {
		pid := process.Pid
		name, _ := process.Name()
		cpu, _ := process.CPUPercent()
		memory_percent, _ := process.MemoryPercent()
		if len(name) != 0 {
			results = append(results, table.Row{fmt.Sprintf("%06s", strconv.Itoa(int(pid))), name, fmt.Sprintf("%05.2f", cpu), fmt.Sprintf("%05.2f", memory_percent)})
		}
	}

	return results
}

func KillProcessByName(process_name string) error {
	pid, _ := strconv.Atoi(process_name)
	process, err := gs.NewProcess(int32(pid))
	if err == nil {
		process.Kill()
	}
	return err
}
