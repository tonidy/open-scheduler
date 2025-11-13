package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/open-scheduler/cli/client"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Display resources",
	Long:  "Display one or many resources",
}

var getNodesCmd = &cobra.Command{
	Use:     "nodes",
	Aliases: []string{"node", "nod"},
	Short:   "List all nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}

		result, err := c.Get("/nodes")
		if err != nil {
			return err
		}

		nodes, ok := result["nodes"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid response format")
		}

		if len(nodes) == 0 {
			fmt.Println("No nodes found")
			return nil
		}

		// Print header
		fmt.Printf("%-35s %-19s %-12s %-10s %-12s\n", "NODE_ID", "LAST_HEARTBEAT", "RAM", "CPU", "DISK")
		fmt.Println(strings.Repeat("-", 95))

		for _, node := range nodes {
			nodeMap := node.(map[string]interface{})
			nodeID := fmt.Sprintf("%v", nodeMap["node_id"])
			// Truncate long node IDs
			if len(nodeID) > 33 {
				nodeID = nodeID[:30] + "..."
			}

			// Format timestamp
			lastHeartbeat := formatTimestamp(nodeMap["last_heartbeat"])

			// Format memory
			ramMB := getFloat64(nodeMap["ram_mb"])
			ramStr := formatMemory(ramMB)

			// Format CPU
			cpuCores := getFloat64(nodeMap["cpu_cores"])
			cpuStr := formatCPU(cpuCores)

			// Format disk
			diskMB := getFloat64(nodeMap["disk_mb"])
			diskStr := formatDisk(diskMB)

			fmt.Printf("%-35s %-19s %-12s %-10s %-12s\n", nodeID, lastHeartbeat, ramStr, cpuStr, diskStr)
		}

		return nil
	},
}

var getJobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "List jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}

		var endpoint string
		active, _ := cmd.Flags().GetBool("active")
		failed, _ := cmd.Flags().GetBool("failed")

		if active {
			endpoint = "/jobs?status=active"
		} else if failed {
			endpoint = "/jobs?status=failed"
		} else {
			endpoint = "/jobs"
		}

		result, err := c.Get(endpoint)
		if err != nil {
			return err
		}

		// Print summary
		if queuedCount, ok := result["queued_count"].(float64); ok {
			fmt.Printf("Queued: %.0f\n", queuedCount)
		}
		if activeCount, ok := result["active_count"].(float64); ok {
			fmt.Printf("Active: %.0f\n", activeCount)
		}
		if completedCount, ok := result["completed_count"].(float64); ok {
			fmt.Printf("Completed: %.0f\n", completedCount)
		}
		if failedCount, ok := result["failed_count"].(float64); ok {
			fmt.Printf("Failed: %.0f\n", failedCount)
		}
		fmt.Println()

		// Print jobs
		if activeJobs, ok := result["active_jobs"].([]interface{}); ok && len(activeJobs) > 0 {
			fmt.Println("\nACTIVE JOBS:")
			fmt.Printf("%-35s %-25s %-15s %-19s\n", "JOB_ID", "NODE_ID", "STATUS", "UPDATED_AT")
			fmt.Println(strings.Repeat("-", 95))
			for _, job := range activeJobs {
				jobMap := job.(map[string]interface{})
				jobID := fmt.Sprintf("%v", jobMap["job_id"])
				if len(jobID) > 33 {
					jobID = jobID[:30] + "..."
				}
				nodeID := fmt.Sprintf("%v", jobMap["node_id"])
				if len(nodeID) > 23 {
					nodeID = nodeID[:20] + "..."
				}
				status := fmt.Sprintf("%v", jobMap["status"])
				updatedAt := formatTimestamp(jobMap["updated_at"])
				fmt.Printf("%-35s %-25s %-15s %-19s\n", jobID, nodeID, status, updatedAt)
			}
			fmt.Println()
		}

		if queuedJobs, ok := result["queued_jobs"].([]interface{}); ok && len(queuedJobs) > 0 {
			fmt.Println("\nQUEUED JOBS:")
			fmt.Printf("%-35s %-50s\n", "JOB_ID", "JOB_NAME")
			fmt.Println(strings.Repeat("-", 90))
			for _, job := range queuedJobs {
				jobMap := job.(map[string]interface{})
				jobID := fmt.Sprintf("%v", jobMap["job_id"])
				if len(jobID) > 33 {
					jobID = jobID[:30] + "..."
				}
				jobName := ""
				if jobData, ok := jobMap["job"].(map[string]interface{}); ok {
					jobName = fmt.Sprintf("%v", jobData["job_name"])
				}
				if len(jobName) > 48 {
					jobName = jobName[:45] + "..."
				}
				fmt.Printf("%-35s %-50s\n", jobID, jobName)
			}
			fmt.Println()
		}

		if completedJobs, ok := result["completed_jobs"].([]interface{}); ok && len(completedJobs) > 0 {
			fmt.Println("\nCOMPLETED JOBS:")
			fmt.Printf("%-35s %-25s %-15s %-19s\n", "JOB_ID", "NODE_ID", "STATUS", "UPDATED_AT")
			fmt.Println(strings.Repeat("-", 95))
			for _, job := range completedJobs {
				jobMap := job.(map[string]interface{})
				jobID := fmt.Sprintf("%v", jobMap["job_id"])
				if len(jobID) > 33 {
					jobID = jobID[:30] + "..."
				}
				nodeID := fmt.Sprintf("%v", jobMap["node_id"])
				if len(nodeID) > 23 {
					nodeID = nodeID[:20] + "..."
				}
				status := fmt.Sprintf("%v", jobMap["status"])
				updatedAt := formatTimestamp(jobMap["updated_at"])
				fmt.Printf("%-35s %-25s %-15s %-19s\n", jobID, nodeID, status, updatedAt)
			}
			fmt.Println()
		}

		if failedJobs, ok := result["failed_jobs"].([]interface{}); ok && len(failedJobs) > 0 {
			fmt.Println("\nFAILED JOBS:")
			fmt.Printf("%-35s %-25s %-15s %-40s\n", "JOB_ID", "NODE_ID", "STATUS", "DETAIL")
			fmt.Println(strings.Repeat("-", 120))
			for _, job := range failedJobs {
				jobMap := job.(map[string]interface{})
				jobID := fmt.Sprintf("%v", jobMap["job_id"])
				if len(jobID) > 33 {
					jobID = jobID[:30] + "..."
				}
				nodeID := fmt.Sprintf("%v", jobMap["node_id"])
				if len(nodeID) > 23 {
					nodeID = nodeID[:20] + "..."
				}
				status := fmt.Sprintf("%v", jobMap["status"])
				detail := fmt.Sprintf("%v", jobMap["detail"])
				if len(detail) > 38 {
					detail = detail[:35] + "..."
				}
				fmt.Printf("%-35s %-25s %-15s %-40s\n", jobID, nodeID, status, detail)
			}
			fmt.Println()
		}

		return nil
	},
}

var getInstancesCmd = &cobra.Command{
	Use:     "instances",
	Aliases: []string{"instance", "inst"},
	Short:   "List all instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}

		result, err := c.Get("/instances")
		if err != nil {
			return err
		}

		instances, ok := result["instances"].([]interface{})
		if !ok {
			return fmt.Errorf("invalid response format")
		}

		if len(instances) == 0 {
			fmt.Println("No instances found")
			return nil
		}

		fmt.Printf("%-35s %-25s %-15s %-40s\n", "JOB_ID", "INSTANCE_ID", "STATUS", "IMAGE")
		fmt.Println(strings.Repeat("-", 120))

		for _, inst := range instances {
			instMap := inst.(map[string]interface{})
			jobID := fmt.Sprintf("%v", instMap["job_id"])
			if len(jobID) > 33 {
				jobID = jobID[:30] + "..."
			}
			instanceID := fmt.Sprintf("%v", instMap["instance_id"])
			if len(instanceID) > 23 {
				instanceID = instanceID[:20] + "..."
			}
			status := fmt.Sprintf("%v", instMap["status"])
			image := fmt.Sprintf("%v", instMap["image_name"])
			if len(image) > 38 {
				image = image[:35] + "..."
			}

			fmt.Printf("%-35s %-25s %-15s %-40s\n", jobID, instanceID, status, image)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getNodesCmd)
	getCmd.AddCommand(getJobsCmd)
	getCmd.AddCommand(getInstancesCmd)

	getJobsCmd.Flags().Bool("active", false, "Show only active jobs")
	getJobsCmd.Flags().Bool("failed", false, "Show only failed jobs")
}

// Helper functions for formatting

func formatTimestamp(timestamp interface{}) string {
	if timestamp == nil {
		return "N/A"
	}

	tsStr := fmt.Sprintf("%v", timestamp)

	// Try to parse ISO 8601 format
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, tsStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	// If parsing fails, try to extract readable part
	if idx := strings.Index(tsStr, "T"); idx > 0 {
		if idx2 := strings.Index(tsStr[idx:], "+"); idx2 > 0 {
			return tsStr[:idx] + " " + tsStr[idx+1:idx+idx2]
		}
		if idx2 := strings.Index(tsStr[idx:], "-"); idx2 > 5 {
			return tsStr[:idx] + " " + tsStr[idx+1:idx+idx2]
		}
	}

	// Fallback: truncate if too long
	if len(tsStr) > 19 {
		return tsStr[:19]
	}
	return tsStr
}

func formatMemory(mb float64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.2f GB", mb/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}

func formatCPU(cores float64) string {
	if cores == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.2f", cores)
}

func formatDisk(mb float64) string {
	if mb >= 1024*1024 {
		return fmt.Sprintf("%.2f TB", mb/(1024*1024))
	}
	if mb >= 1024 {
		return fmt.Sprintf("%.2f GB", mb/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}

func getFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
