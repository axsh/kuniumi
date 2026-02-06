package kuniumi

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func (a *App) buildContainerizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "containerize",
		Short: "Build a Docker image of this application",
		RunE: func(cmd *cobra.Command, args []string) error {
			imageName, _ := cmd.Flags().GetString("image")
			push, _ := cmd.Flags().GetBool("push")

			if imageName == "" {
				return fmt.Errorf("image name is required")
			}

			// Generate Dockerfile
			dockerfile := `
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/app .
ENTRYPOINT ["./app"]
`
			// Create a temporary file for the Dockerfile
			tmpFile, err := os.CreateTemp("", "dockerfile-kuniumi-*")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			dockerfilename := tmpFile.Name()
			defer os.Remove(dockerfilename)

			if _, err := tmpFile.Write([]byte(dockerfile)); err != nil {
				tmpFile.Close()
				return fmt.Errorf("failed to write dockerfile: %w", err)
			}
			if err := tmpFile.Close(); err != nil {
				return fmt.Errorf("failed to close temp file: %w", err)
			}

			// Build
			fmt.Printf("Building image %s...\n", imageName)
			buildCmd := exec.Command("docker", "build", "-f", dockerfilename, "-t", imageName, ".")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return err
			}

			// Push
			if push {
				fmt.Printf("Pushing image %s...\n", imageName)
				pushCmd := exec.Command("docker", "push", imageName)
				pushCmd.Stdout = os.Stdout
				pushCmd.Stderr = os.Stderr
				if err := pushCmd.Run(); err != nil {
					return err
				}
			}

			fmt.Println("Done.")
			return nil
		},
	}
	cmd.Flags().String("image", "", "Docker image name (e.g. my-app:latest)")
	cmd.Flags().Bool("push", false, "Push image after build")
	return cmd
}
