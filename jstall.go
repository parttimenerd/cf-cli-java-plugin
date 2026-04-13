/*
 * Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved.
 * This file is licensed under the Apache Software License, v. 2 except as noted
 * otherwise in the LICENSE file at the root of the repository.
 */

package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/shlex"
)

//go:embed dist/jstall-minimal.jar
var jstallJarBytes []byte

func javaExecutable() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func getJavaMajorVersion(javaPath string) (int, error) {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}
	outputStr := string(output)
	start := strings.Index(outputStr, "\"")
	if start == -1 {
		return 0, fmt.Errorf("cannot parse java version output")
	}
	end := strings.Index(outputStr[start+1:], "\"")
	if end == -1 {
		return 0, fmt.Errorf("cannot parse java version output")
	}
	versionStr := outputStr[start+1 : start+1+end]
	parts := strings.Split(versionStr, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	// Old format: 1.X.Y (Java 8 and below)
	if major == 1 && len(parts) > 1 {
		major, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
	}
	return major, nil
}

func platformJavaCandidates() []string {
	exe := javaExecutable()
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		matches, _ := filepath.Glob("/Library/Java/JavaVirtualMachines/*/Contents/Home/bin/" + exe)
		candidates = append(candidates, matches...)
		matches, _ = filepath.Glob("/opt/homebrew/Cellar/openjdk/*/bin/" + exe)
		candidates = append(candidates, matches...)
		matches, _ = filepath.Glob("/opt/homebrew/Cellar/openjdk@*/*/bin/" + exe)
		candidates = append(candidates, matches...)
	case "linux":
		matches, _ := filepath.Glob("/usr/lib/jvm/*/bin/" + exe)
		candidates = append(candidates, matches...)
	case "windows":
		for _, envVar := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"} {
			base := os.Getenv(envVar)
			if base == "" {
				continue
			}
			for _, vendor := range []string{"Java", "SapMachine", "Eclipse Adoptium", "Microsoft", "Amazon Corretto", "Zulu"} {
				matches, _ := filepath.Glob(filepath.Join(base, vendor, "*", "bin", exe))
				candidates = append(candidates, matches...)
			}
		}
	}
	return candidates
}

func findJava17Plus() (string, error) {
	var candidates []string
	exe := javaExecutable()
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		candidates = append(candidates, filepath.Join(javaHome, "bin", exe))
	}
	if path, err := exec.LookPath(exe); err == nil {
		candidates = append(candidates, path)
	}
	candidates = append(candidates, platformJavaCandidates()...)

	seen := make(map[string]bool)
	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			resolved = candidate
		}
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		if version, err := getJavaMajorVersion(candidate); err == nil && version >= 17 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no Java 17+ installation found. Install a JDK 17+ and ensure it is on your PATH or set JAVA_HOME")
}

func jstallJarHash() string {
	h := sha256.Sum256(jstallJarBytes)
	return hex.EncodeToString(h[:])
}

func ensureJstallJar() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	pluginCacheDir := filepath.Join(cacheDir, "cf-java-plugin")
	if err := os.MkdirAll(pluginCacheDir, 0o755); err != nil {
		return "", err
	}
	jarPath := filepath.Join(pluginCacheDir, "jstall-minimal.jar")
	hashPath := jarPath + ".sha256"

	// Check if cached JAR matches the embedded version by SHA-256 hash
	expectedHash := jstallJarHash()
	if cachedHash, err := os.ReadFile(hashPath); err == nil && string(cachedHash) == expectedHash {
		if _, err := os.Stat(jarPath); err == nil {
			return jarPath, nil
		}
	}

	// Extract embedded JAR and write hash
	if err := os.WriteFile(jarPath, jstallJarBytes, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(hashPath, []byte(expectedHash), 0o644); err != nil {
		// Non-fatal: JAR is already written, just can't cache the hash
		_ = err
	}
	return jarPath, nil
}

func quoteArgForDisplay(arg string) string {
	if arg == "" || strings.ContainsAny(arg, " \t\n\"'\\") {
		return strconv.Quote(arg)
	}
	return arg
}

func formatCommandForDisplay(command string, args []string) string {
	displayArgs := make([]string, len(args))
	for i, arg := range args {
		displayArgs[i] = quoteArgForDisplay(arg)
	}
	return command + " " + strings.Join(displayArgs, " ")
}

func (c *JavaPlugin) executeJstall(appName string, jstallArgs string, appInstanceIndex int, dryRun bool) (string, error) {
	javaPath, err := findJava17Plus()
	if err != nil {
		return "", err
	}
	c.logVerbosef("Found Java 17+: %s", javaPath)

	jarPath, err := ensureJstallJar()
	if err != nil {
		return "", fmt.Errorf("failed to extract jstall JAR: %w", err)
	}
	c.logVerbosef("JStall JAR at: %s", jarPath)

	args := []string{"-jar", jarPath}

	// Build SSH command with PATH setup so jps/jcmd are discoverable on remote container
	// SAP Java Buildpack puts JDK tools at deep paths not on $PATH
	pathSetup := `JDK_BIN=$(dirname "$(find . -executable -name jps 2>/dev/null | head -1)" 2>/dev/null); if [ -n "$JDK_BIN" ]; then export PATH="$JDK_BIN:$PATH"; fi;`
	sshCmd := "cf ssh " + appName
	if appInstanceIndex != -1 {
		sshCmd += " --app-instance-index " + strconv.Itoa(appInstanceIndex)
	}
	sshCmd += " -c"
	args = append(args, "--ssh", sshCmd)
	args = append(args, "--ssh-prefix", pathSetup)

	if jstallArgs != "" {
		splitArgs, err := shlex.Split(jstallArgs)
		if err != nil {
			return "", fmt.Errorf("invalid jstall arguments: %w", err)
		}
		args = append(args, splitArgs...)
	}

	displayCmd := formatCommandForDisplay(javaPath, args)
	c.logVerbosef("JStall command: %s", displayCmd)

	if dryRun {
		return displayCmd, nil
	}

	// Pre-validate SSH connectivity to avoid confusing "No JVMs found" errors
	if appName != "" {
		testArgs := []string{"ssh", appName}
		if appInstanceIndex != -1 {
			testArgs = append(testArgs, "--app-instance-index", strconv.Itoa(appInstanceIndex))
		}
		testArgs = append(testArgs, "-c", "echo ok")
		testCmd := exec.Command("cf", testArgs...)
		testOutput, testErr := testCmd.CombinedOutput()
		if testErr != nil {
			outputStr := strings.TrimSpace(string(testOutput))
			if outputStr != "" {
				return "", fmt.Errorf("cannot connect to application via SSH: %s", outputStr)
			}
			return "", fmt.Errorf("cannot connect to application via SSH: %w", testErr)
		}
	}

	cmd := exec.Command(javaPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("jstall execution failed: %w", err)
	}
	return "", nil
}
