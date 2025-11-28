// Package main runs all E2E test combinations in parallel goroutines.
//
// This script:
// 1. Builds DEB and RPM packages in parallel
// 2. Runs E2E tests for all distribution and database combinations in parallel:
//   - Debian + MariaDB
//   - Debian + MySQL
//   - Rocky + MariaDB
//   - Rocky + MySQL
//
// All tests run in parallel and continue even if one fails.
// Output is written to both console and a shared log file.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ANSI color codes
const (
	ColorBlue    = "\033[94m"
	ColorGreen   = "\033[92m"
	ColorYellow  = "\033[93m"
	ColorRed     = "\033[91m"
	ColorCyan    = "\033[96m"
	ColorMagenta = "\033[95m"
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
)

// BuildTarget represents a package build target
type BuildTarget struct {
	Name       string
	MakeTarget string
	Color      string
}

// TestCombination represents a test environment combination
type TestCombination struct {
	Name        string
	MakeTarget  string
	Color       string
	ProjectName string
}

// TestResult holds the result of a build or test run
type TestResult struct {
	Name       string
	Success    bool
	ReturnCode int
	Duration   time.Duration
}

// TestRunner manages parallel execution of builds and tests
type TestRunner struct {
	logFile     *os.File
	logMutex    sync.Mutex
	results     map[string]*TestResult
	resultMutex sync.Mutex
	workDir     string

	builds       []BuildTarget
	combinations []TestCombination
}

// NewTestRunner creates a new test runner
func NewTestRunner(logPath string) (*TestRunner, error) {
	// Create log directory if needed
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Get working directory (app root)
	workDir, err := os.Getwd()
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Navigate to app root (parent of tests/e2e)
	workDir = filepath.Dir(filepath.Dir(workDir))

	return &TestRunner{
		logFile: logFile,
		results: make(map[string]*TestResult),
		workDir: workDir,
		builds: []BuildTarget{
			{Name: "DEB-BUILD", MakeTarget: "package-deb-docker", Color: ColorBlue},
			{Name: "RPM-BUILD", MakeTarget: "package-rpm-docker", Color: ColorCyan},
		},
		combinations: []TestCombination{
			{Name: "DEB-MARIADB", MakeTarget: "e2e-test-debian-mariadb-quick", Color: ColorBlue, ProjectName: "dbcalm-e2e-go-deb-mariadb"},
			{Name: "DEB-MYSQL", MakeTarget: "e2e-test-debian-mysql-quick", Color: ColorGreen, ProjectName: "dbcalm-e2e-go-deb-mysql"},
			{Name: "ROCKY-MARIADB", MakeTarget: "e2e-test-rocky-mariadb-quick", Color: ColorCyan, ProjectName: "dbcalm-e2e-go-rocky-mariadb"},
			{Name: "ROCKY-MYSQL", MakeTarget: "e2e-test-rocky-mysql-quick", Color: ColorMagenta, ProjectName: "dbcalm-e2e-go-rocky-mysql"},
		},
	}, nil
}

// Close closes the log file
func (r *TestRunner) Close() {
	if r.logFile != nil {
		r.logFile.Close()
	}
}

// log writes a message to both console and log file
func (r *TestRunner) log(prefix, message, color string) {
	r.logMutex.Lock()
	defer r.logMutex.Unlock()

	formatted := fmt.Sprintf("[%s] %s", prefix, message)

	// Console output with color
	if color != "" {
		fmt.Printf("%s%s%s\n", color, formatted, ColorReset)
	} else {
		fmt.Println(formatted)
	}

	// Log file output (no color)
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	fmt.Fprintf(r.logFile, "%s %s\n", timestamp, formatted)
}

// copyArtifacts copies built packages from build/dist to tests/e2e/artifacts
func (r *TestRunner) copyArtifacts() error {
	srcDir := filepath.Join(r.workDir, "build", "dist")
	dstDir := filepath.Join(r.workDir, "tests", "e2e", "artifacts")

	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Copy .deb files
	debFiles, _ := filepath.Glob(filepath.Join(srcDir, "*.deb"))
	for _, src := range debFiles {
		dst := filepath.Join(dstDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", filepath.Base(src), err)
		}
		r.log("ARTIFACTS", fmt.Sprintf("Copied %s to artifacts/", filepath.Base(src)), ColorYellow)
	}

	// Copy .rpm files
	rpmFiles, _ := filepath.Glob(filepath.Join(srcDir, "*.rpm"))
	for _, src := range rpmFiles {
		dst := filepath.Join(dstDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", filepath.Base(src), err)
		}
		r.log("ARTIFACTS", fmt.Sprintf("Copied %s to artifacts/", filepath.Base(src)), ColorYellow)
	}

	if len(debFiles) == 0 && len(rpmFiles) == 0 {
		return fmt.Errorf("no packages found in %s", srcDir)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// cleanupDockerCompose cleans up Docker Compose containers and volumes
func (r *TestRunner) cleanupDockerCompose(projectName string) {
	cmd := exec.Command("docker", "compose", "-p", projectName, "down", "-v")
	cmd.Dir = filepath.Join(r.workDir, "tests", "e2e", "common")
	_ = cmd.Run() // Ignore errors - cleanup is best effort
}

// runMakeTarget runs a make target and streams output
func (r *TestRunner) runMakeTarget(name, makeTarget, color, projectName string) {
	r.log(name, fmt.Sprintf("Starting (make %s)", makeTarget), color)
	startTime := time.Now()

	cmd := exec.Command("make", makeTarget)
	cmd.Dir = r.workDir

	// Create pipe for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.storeResult(name, false, -1, time.Since(startTime))
		r.log(name, fmt.Sprintf("✗ ERROR: failed to create pipe: %v", err), ColorRed+ColorBold)
		return
	}
	cmd.Stderr = cmd.Stdout // Merge stderr into stdout

	if err := cmd.Start(); err != nil {
		r.storeResult(name, false, -1, time.Since(startTime))
		r.log(name, fmt.Sprintf("✗ ERROR: failed to start: %v", err), ColorRed+ColorBold)
		return
	}

	// Stream output
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			r.log(name, line, color)
		}
	}

	// Wait for completion
	err = cmd.Wait()
	duration := time.Since(startTime)
	durationStr := formatDuration(duration)

	var success bool
	var returnCode int

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			returnCode = exitErr.ExitCode()
		} else {
			returnCode = -1
		}
		success = false
		r.log(name, fmt.Sprintf("✗ FAILED with exit code %d (duration: %s)", returnCode, durationStr), ColorRed+ColorBold)
	} else {
		success = true
		returnCode = 0
		r.log(name, fmt.Sprintf("✓ PASSED (duration: %s)", durationStr), ColorGreen+ColorBold)
	}

	// Cleanup docker compose if this was a test
	if projectName != "" {
		r.log(name, fmt.Sprintf("Cleaning up Docker containers (project: %s)", projectName), color)
		r.cleanupDockerCompose(projectName)
	}

	r.storeResult(name, success, returnCode, duration)
}

// storeResult stores a test result
func (r *TestRunner) storeResult(name string, success bool, returnCode int, duration time.Duration) {
	r.resultMutex.Lock()
	defer r.resultMutex.Unlock()

	r.results[name] = &TestResult{
		Name:       name,
		Success:    success,
		ReturnCode: returnCode,
		Duration:   duration,
	}
}

// runBuilds runs package builds in parallel
func (r *TestRunner) runBuilds() bool {
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("%sPhase 1: Building Packages%s\n", ColorBold, ColorReset)
	fmt.Println("================================================================================")
	fmt.Println()

	var wg sync.WaitGroup
	for _, build := range r.builds {
		wg.Add(1)
		go func(b BuildTarget) {
			defer wg.Done()
			r.runMakeTarget(b.Name, b.MakeTarget, b.Color, "")
		}(build)
	}
	wg.Wait()

	// Check if all builds passed
	allPassed := true
	for _, build := range r.builds {
		if result, ok := r.results[build.Name]; ok && !result.Success {
			allPassed = false
			break
		}
	}

	if allPassed {
		fmt.Println()
		fmt.Printf("%s%s✓ All builds completed successfully%s\n", ColorGreen, ColorBold, ColorReset)
	} else {
		fmt.Println()
		fmt.Printf("%s%s✗ Some builds failed%s\n", ColorRed, ColorBold, ColorReset)
	}

	return allPassed
}

// runTests runs test combinations in parallel
func (r *TestRunner) runTests(selectedTests map[string]bool) bool {
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("%sPhase 2: Running Tests%s\n", ColorBold, ColorReset)
	fmt.Println("================================================================================")
	fmt.Println()

	var wg sync.WaitGroup
	for _, combo := range r.combinations {
		// Skip if not in selected tests (when filtering)
		if selectedTests != nil && !selectedTests[combo.Name] {
			continue
		}

		wg.Add(1)
		go func(c TestCombination) {
			defer wg.Done()
			r.runMakeTarget(c.Name, c.MakeTarget, c.Color, c.ProjectName)
		}(combo)
	}
	wg.Wait()

	// Check if all tests passed
	allPassed := true
	for _, combo := range r.combinations {
		if selectedTests != nil && !selectedTests[combo.Name] {
			continue
		}
		if result, ok := r.results[combo.Name]; ok && !result.Success {
			allPassed = false
			break
		}
	}

	return allPassed
}

// printFailedTestLogs extracts and prints logs from failed test containers
func (r *TestRunner) printFailedTestLogs() {
	var failedTests []TestCombination
	for _, combo := range r.combinations {
		if result, ok := r.results[combo.Name]; ok && !result.Success {
			failedTests = append(failedTests, combo)
		}
	}

	if len(failedTests) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("%sContainer Logs for Failed Tests%s\n", ColorBold, ColorReset)
	fmt.Println("================================================================================")

	for _, test := range failedTests {
		fmt.Println()
		fmt.Printf("%s%s=== Logs for %s ===%s\n", ColorRed, ColorBold, test.Name, ColorReset)
		fmt.Println()

		cmd := exec.Command("docker", "compose", "-p", test.ProjectName, "logs", "--no-color")
		cmd.Dir = filepath.Join(r.workDir, "tests", "e2e", "common")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			fmt.Print(string(output))
		} else {
			fmt.Printf("%sNo logs available for %s%s\n", ColorYellow, test.Name, ColorReset)
		}
	}

	fmt.Println("================================================================================")
}

// printSummary prints the final summary
func (r *TestRunner) printSummary(selectedTests map[string]bool) {
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("%sFinal Summary%s\n", ColorBold, ColorReset)
	fmt.Println("================================================================================")

	// Print build results
	fmt.Printf("\n%sBuilds:%s\n", ColorBold, ColorReset)
	for _, build := range r.builds {
		if result, ok := r.results[build.Name]; ok {
			statusColor := ColorGreen
			statusSymbol := "✓"
			statusText := "PASSED"
			if !result.Success {
				statusColor = ColorRed
				statusSymbol = "✗"
				statusText = "FAILED"
			}
			fmt.Printf("  %s%s %-15s %-8s (duration: %s)%s\n",
				statusColor, statusSymbol, result.Name, statusText, formatDuration(result.Duration), ColorReset)
		}
	}

	// Print test results
	fmt.Printf("\n%sTests:%s\n", ColorBold, ColorReset)
	for _, combo := range r.combinations {
		if selectedTests != nil && !selectedTests[combo.Name] {
			continue
		}
		if result, ok := r.results[combo.Name]; ok {
			statusColor := ColorGreen
			statusSymbol := "✓"
			statusText := "PASSED"
			if !result.Success {
				statusColor = ColorRed
				statusSymbol = "✗"
				statusText = "FAILED"
			}
			fmt.Printf("  %s%s %-15s %-8s (duration: %s)%s\n",
				statusColor, statusSymbol, result.Name, statusText, formatDuration(result.Duration), ColorReset)
		}
	}

	fmt.Println("================================================================================")

	// Write summary to log file
	r.logMutex.Lock()
	fmt.Fprintln(r.logFile)
	fmt.Fprintln(r.logFile, "================================================================================")
	fmt.Fprintln(r.logFile, "Final Summary")
	fmt.Fprintln(r.logFile, "================================================================================")
	fmt.Fprintln(r.logFile, "\nBuilds:")
	for _, build := range r.builds {
		if result, ok := r.results[build.Name]; ok {
			status := "PASSED"
			if !result.Success {
				status = "FAILED"
			}
			fmt.Fprintf(r.logFile, "  %s - %s (duration: %s, exit code: %d)\n",
				status, result.Name, formatDuration(result.Duration), result.ReturnCode)
		}
	}
	fmt.Fprintln(r.logFile, "\nTests:")
	for _, combo := range r.combinations {
		if selectedTests != nil && !selectedTests[combo.Name] {
			continue
		}
		if result, ok := r.results[combo.Name]; ok {
			status := "PASSED"
			if !result.Success {
				status = "FAILED"
			}
			fmt.Fprintf(r.logFile, "  %s - %s (duration: %s, exit code: %d)\n",
				status, result.Name, formatDuration(result.Duration), result.ReturnCode)
		}
	}
	fmt.Fprintln(r.logFile, "================================================================================")
	fmt.Fprintf(r.logFile, "Completed: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"))
	r.logMutex.Unlock()
}

// RunAll runs builds and tests in sequence
func (r *TestRunner) RunAll(selectedTests map[string]bool) bool {
	// Write log file header
	r.logMutex.Lock()
	fmt.Fprintln(r.logFile, "================================================================================")
	fmt.Fprintln(r.logFile, "E2E Test Run - Build and Test All Combinations (Go)")
	fmt.Fprintf(r.logFile, "Started: %s\n", time.Now().UTC().Format("2006-01-02 15:04:05"))
	fmt.Fprintln(r.logFile, "================================================================================")
	fmt.Fprintln(r.logFile)
	r.logMutex.Unlock()

	fmt.Printf("%sStarting E2E build and test process...%s\n", ColorBold, ColorReset)
	fmt.Printf("Log file: %s\n", r.logFile.Name())

	// Phase 1: Build packages in parallel
	buildsPassed := r.runBuilds()

	if !buildsPassed {
		fmt.Println()
		fmt.Printf("%s%sBuilds failed. Skipping tests.%s\n", ColorRed, ColorBold, ColorReset)
		return false
	}

	// Copy artifacts to test directory
	fmt.Println()
	fmt.Printf("%sCopying artifacts to tests/e2e/artifacts/%s\n", ColorBold, ColorReset)
	if err := r.copyArtifacts(); err != nil {
		fmt.Printf("%s%sFailed to copy artifacts: %v%s\n", ColorRed, ColorBold, err, ColorReset)
		return false
	}

	// Phase 2: Run tests in parallel
	testsPassed := r.runTests(selectedTests)

	// Print container logs for failed tests
	r.printFailedTestLogs()

	// Print final summary
	r.printSummary(selectedTests)

	allPassed := buildsPassed && testsPassed
	if allPassed {
		fmt.Printf("\n%s%sAll builds and tests passed!%s\n", ColorGreen, ColorBold, ColorReset)
	} else {
		fmt.Printf("\n%s%sSome builds or tests failed. Check log file for details.%s\n", ColorRed, ColorBold, ColorReset)
	}

	return allPassed
}

// formatDuration formats a duration in HH:MM:SS format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

func main() {
	// Parse command line flags
	debianMariaDB := flag.Bool("debian-mariadb", false, "Run only Debian + MariaDB tests")
	debianMySQL := flag.Bool("debian-mysql", false, "Run only Debian + MySQL tests")
	rockyMariaDB := flag.Bool("rocky-mariadb", false, "Run only Rocky + MariaDB tests")
	rockyMySQL := flag.Bool("rocky-mysql", false, "Run only Rocky + MySQL tests")
	logDir := flag.String("log-dir", "", "Directory for log files (default: tests/e2e/logs)")
	flag.Parse()

	// Determine log directory
	if *logDir == "" {
		// Default to tests/e2e/logs relative to script location
		execPath, _ := os.Executable()
		*logDir = filepath.Join(filepath.Dir(execPath), "logs")
		// If running with go run, use current directory
		if _, err := os.Stat(*logDir); os.IsNotExist(err) {
			*logDir = "logs"
		}
	}

	// Create log file with timestamp
	timestamp := time.Now().UTC().Format("20060102-150405")
	logPath := filepath.Join(*logDir, fmt.Sprintf("all-tests-%s.log", timestamp))

	// Determine which tests to run
	var selectedTests map[string]bool
	if *debianMariaDB || *debianMySQL || *rockyMariaDB || *rockyMySQL {
		selectedTests = make(map[string]bool)
		if *debianMariaDB {
			selectedTests["DEB-MARIADB"] = true
		}
		if *debianMySQL {
			selectedTests["DEB-MYSQL"] = true
		}
		if *rockyMariaDB {
			selectedTests["ROCKY-MARIADB"] = true
		}
		if *rockyMySQL {
			selectedTests["ROCKY-MYSQL"] = true
		}
	}

	// Create test runner
	runner, err := NewTestRunner(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close()

	// Run builds and tests
	if runner.RunAll(selectedTests) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
