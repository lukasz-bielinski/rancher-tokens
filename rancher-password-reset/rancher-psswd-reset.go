package rancher_password_reset

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func ResetRancherPassword() (string, error) {
	// Buffers for first command
	var stdout1, stderr1 bytes.Buffer

	// Get the pod name
	cmd1 := exec.Command("kubectl", "-n", "cattle-system", "get", "pods", "-l", "app=rancher", "--no-headers")
	cmd1.Stdout = &stdout1
	cmd1.Stderr = &stderr1
	if err := cmd1.Run(); err != nil {
		log.Fatalf("cmd1.Run() failed: %s: %s", stderr1.String(), err)
	}

	podNames := strings.Fields(stdout1.String())
	if len(podNames) == 0 {
		log.Fatal("No Rancher pods found")
	}
	podName := podNames[0] // Take the first pod name from the list

	// Buffers for second command
	var stdout2, stderr2 bytes.Buffer

	// Reset the Rancher password
	cmd2 := exec.Command("kubectl", "-n", "cattle-system", "exec", podName, "-c", "rancher", "--", "reset-password")
	cmd2.Stdout = &stdout2
	cmd2.Stderr = &stderr2
	if err := cmd2.Run(); err != nil {
		log.Fatalf("cmd2.Run() failed: %s: %s", stderr2.String(), err)
	}

	outputLines := strings.Split(stdout2.String(), "\n")
	for i, line := range outputLines {
		if strings.HasPrefix(line, "New password for default admin user") {
			if i+1 < len(outputLines) {
				password := strings.TrimSpace(outputLines[i+1])
				fmt.Printf("Password reset successfully for pod %s: %s\n", podName, password)
				return password, nil
			}
		}
	}
	return "", fmt.Errorf("could not extract the new password from the output")
}
