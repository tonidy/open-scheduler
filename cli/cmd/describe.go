package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/open-scheduler/cli/client"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show detailed information about a resource",
	Long:  "Show detailed information about a specific resource",
}

var describeNodeCmd = &cobra.Command{
	Use:     "node NODE_NAME",
	Aliases: []string{"nod", "nodes"},
	Short:   "Describe a node",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID := args[0]
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}
		
		result, err := c.Get(fmt.Sprintf("/nodes/%s", nodeID))
		if err != nil {
			return err
		}
		
		// Format node information
		fmt.Println("Name:         ", result["node_id"])
		fmt.Println("Last Heartbeat:", formatTimestamp(result["last_heartbeat"]))
		
		ramMB := getFloat64(result["ram_mb"])
		cpuCores := getFloat64(result["cpu_cores"])
		diskMB := getFloat64(result["disk_mb"])
		
		fmt.Println("\nResources:")
		fmt.Println("  RAM:        ", formatMemory(ramMB))
		fmt.Println("  CPU Cores:  ", formatCPU(cpuCores))
		fmt.Println("  Disk:       ", formatDisk(diskMB))
		
		if metadata, ok := result["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
			fmt.Println("\nMetadata:")
			for k, v := range metadata {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
		
		return nil
	},
}

var describeJobCmd = &cobra.Command{
	Use:     "job JOB_ID",
	Aliases: []string{"jobs"},
	Short:   "Describe a job",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}
		
		result, err := c.Get(fmt.Sprintf("/jobs/%s", jobID))
		if err != nil {
			return err
		}
		
		// Basic Information
		fmt.Println("Name:         ", result["job_id"])
		fmt.Println("Status:       ", result["status"])
		if nodeID, ok := result["node_id"].(string); ok && nodeID != "" {
			fmt.Println("Node:         ", nodeID)
		}
		if detail, ok := result["detail"].(string); ok && detail != "" {
			fmt.Println("Detail:       ", detail)
		}
		if claimedAt := result["claimed_at"]; claimedAt != nil {
			fmt.Println("Claimed At:   ", formatTimestamp(claimedAt))
		}
		if updatedAt := result["updated_at"]; updatedAt != nil {
			fmt.Println("Updated At:   ", formatTimestamp(updatedAt))
		}
		
		// Job Specification
		if job, ok := result["job"].(map[string]interface{}); ok {
			fmt.Println("\nJob Specification:")
			if jobName, ok := job["job_name"].(string); ok && jobName != "" {
				fmt.Println("  Name:            ", jobName)
			}
			if jobType, ok := job["job_type"].(string); ok && jobType != "" {
				fmt.Println("  Type:            ", jobType)
			}
			if driverType, ok := job["driver_type"].(string); ok && driverType != "" {
				fmt.Println("  Driver:          ", driverType)
			}
			if workloadType, ok := job["workload_type"].(string); ok && workloadType != "" {
				fmt.Println("  Workload Type:   ", workloadType)
			}
			if command, ok := job["command"].(string); ok && command != "" {
				fmt.Println("  Command:         ", command)
			}
			
			if selectedClusters, ok := job["selected_clusters"].([]interface{}); ok && len(selectedClusters) > 0 {
				fmt.Print("  Selected Clusters: ")
				clusters := make([]string, 0, len(selectedClusters))
				for _, c := range selectedClusters {
					clusters = append(clusters, fmt.Sprintf("%v", c))
				}
				fmt.Println(strings.Join(clusters, ", "))
			}
			
			// Instance Config
			if instanceConfig, ok := job["instance_config"].(map[string]interface{}); ok {
				fmt.Println("\n  Instance Config:")
				if imageName, ok := instanceConfig["image_name"].(string); ok && imageName != "" {
					fmt.Println("    Image:         ", imageName)
				}
				if entrypoint, ok := instanceConfig["entrypoint"].([]interface{}); ok && len(entrypoint) > 0 {
					epStrs := make([]string, 0, len(entrypoint))
					for _, e := range entrypoint {
						epStrs = append(epStrs, fmt.Sprintf("%v", e))
					}
					fmt.Println("    Entrypoint:    ", strings.Join(epStrs, " "))
				}
				if arguments, ok := instanceConfig["arguments"].([]interface{}); ok && len(arguments) > 0 {
					argStrs := make([]string, 0, len(arguments))
					for _, a := range arguments {
						argStrs = append(argStrs, fmt.Sprintf("%v", a))
					}
					fmt.Println("    Arguments:     ", strings.Join(argStrs, " "))
				}
			}
			
			// Resources
			if resources, ok := job["resource_requirements"].(map[string]interface{}); ok {
				fmt.Println("\n  Resources:")
				if memoryLimitMB := getFloat64(resources["memory_limit_mb"]); memoryLimitMB > 0 {
					fmt.Println("    Memory Limit:  ", formatMemory(memoryLimitMB))
				}
				if memoryReservedMB := getFloat64(resources["memory_reserved_mb"]); memoryReservedMB > 0 {
					fmt.Println("    Memory Reserved:", formatMemory(memoryReservedMB))
				}
				if cpuLimitCores := getFloat64(resources["cpu_limit_cores"]); cpuLimitCores > 0 {
					fmt.Println("    CPU Limit:      ", formatCPU(cpuLimitCores), "cores")
				}
				if cpuReservedCores := getFloat64(resources["cpu_reserved_cores"]); cpuReservedCores > 0 {
					fmt.Println("    CPU Reserved:  ", formatCPU(cpuReservedCores), "cores")
				}
			}
			
			// Volumes
			if volumes, ok := job["volume_mounts"].([]interface{}); ok && len(volumes) > 0 {
				fmt.Println("\n  Volume Mounts:")
				for i, vol := range volumes {
					if volMap, ok := vol.(map[string]interface{}); ok {
						source := fmt.Sprintf("%v", volMap["source_path"])
						target := fmt.Sprintf("%v", volMap["target_path"])
						readOnly := ""
						if ro, ok := volMap["read_only"].(bool); ok && ro {
							readOnly = " (read-only)"
						}
						fmt.Printf("    %d: %s -> %s%s\n", i+1, source, target, readOnly)
					}
				}
			}
			
			// Environment Variables
			if envVars, ok := job["environment_variables"].(map[string]interface{}); ok && len(envVars) > 0 {
				fmt.Println("\n  Environment Variables:")
				for k, v := range envVars {
					fmt.Printf("    %s=%v\n", k, v)
				}
			}
			
			// Metadata
			if metadata, ok := job["job_metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
				fmt.Println("\n  Metadata:")
				for k, v := range metadata {
					fmt.Printf("    %s: %v\n", k, v)
				}
			}
			
			// Retry Info
			if retryCount, ok := job["retry_count"].(float64); ok {
				fmt.Println("\n  Retry Information:")
				fmt.Printf("    Retry Count:   %.0f\n", retryCount)
				if maxRetries, ok := job["max_retries"].(float64); ok {
					fmt.Printf("    Max Retries:   %.0f\n", maxRetries)
				}
			}
		}
		
		// Events
		if events, ok := result["events"].([]interface{}); ok && len(events) > 0 {
			fmt.Println("\nEvents:")
			for i, event := range events {
				if i >= 10 { // Limit to last 10 events
					fmt.Printf("  ... and %d more events\n", len(events)-10)
					break
				}
				eventStr := fmt.Sprintf("%v", event)
				// Format event timestamp if present
				if idx := strings.Index(eventStr, "] "); idx > 0 {
					tsPart := eventStr[:idx+1]
					msgPart := eventStr[idx+2:]
					// Try to parse and reformat timestamp
					tsPart = strings.Trim(tsPart, "[]")
					if t, err := time.Parse(time.RFC3339Nano, tsPart); err == nil {
						tsPart = t.Format("2006-01-02 15:04:05")
					}
					fmt.Printf("  [%s] %s\n", tsPart, msgPart)
				} else {
					fmt.Printf("  %s\n", eventStr)
				}
			}
		}
		
		return nil
	},
}

var describeInstanceCmd = &cobra.Command{
	Use:     "instance JOB_ID",
	Aliases: []string{"inst", "instances"},
	Short:   "Describe an instance",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}
		
		result, err := c.Get(fmt.Sprintf("/instances/%s", jobID))
		if err != nil {
			return err
		}
		
		fmt.Println("Job ID:       ", result["job_id"])
		
		if instanceData, ok := result["instance_data"].(map[string]interface{}); ok {
			fmt.Println("\nInstance Information:")
			if instanceID, ok := instanceData["instance_id"].(string); ok && instanceID != "" {
				fmt.Println("  Instance ID:    ", instanceID)
			}
			if instanceName, ok := instanceData["instance_name"].(string); ok && instanceName != "" {
				fmt.Println("  Name:           ", instanceName)
			}
			if status, ok := instanceData["status"].(string); ok && status != "" {
				fmt.Println("  Status:         ", status)
			}
			image := ""
			if img, ok := instanceData["image_name"].(string); ok && img != "" {
				image = img
				fmt.Println("  Image:          ", image)
			}
			if fullImage, ok := instanceData["image"].(string); ok && fullImage != "" && fullImage != image {
				fmt.Println("  Full Image:     ", fullImage)
			}
			
			if pid, ok := instanceData["pid"].(float64); ok && pid > 0 {
				fmt.Printf("  PID:            %.0f\n", pid)
			}
			if exitCode, ok := instanceData["exit_code"].(float64); ok {
				fmt.Printf("  Exit Code:      %.0f\n", exitCode)
			}
			
			if created, ok := instanceData["created"].(string); ok && created != "" {
				fmt.Println("  Created:        ", formatTimestamp(created))
			}
			if startedAt, ok := instanceData["started_at"].(string); ok && startedAt != "" {
				fmt.Println("  Started At:     ", formatTimestamp(startedAt))
			}
			if finishedAt, ok := instanceData["finished_at"].(string); ok && finishedAt != "" {
				fmt.Println("  Finished At:    ", formatTimestamp(finishedAt))
			}
			
			// Command and Arguments
			if command, ok := instanceData["command"].([]interface{}); ok && len(command) > 0 {
				cmdStrs := make([]string, 0, len(command))
				for _, c := range command {
					cmdStrs = append(cmdStrs, fmt.Sprintf("%v", c))
				}
				fmt.Println("  Command:        ", strings.Join(cmdStrs, " "))
			}
			if args, ok := instanceData["args"].([]interface{}); ok && len(args) > 0 {
				argStrs := make([]string, 0, len(args))
				for _, a := range args {
					argStrs = append(argStrs, fmt.Sprintf("%v", a))
				}
				fmt.Println("  Arguments:      ", strings.Join(argStrs, " "))
			}
			
			// Ports
			if ports, ok := instanceData["ports"].([]interface{}); ok && len(ports) > 0 {
				fmt.Println("\n  Ports:")
				for _, p := range ports {
					fmt.Printf("    - %v\n", p)
				}
			}
			
			// Volumes
			if volumes, ok := instanceData["volumes"].([]interface{}); ok && len(volumes) > 0 {
				fmt.Println("\n  Volumes:")
				for _, v := range volumes {
					fmt.Printf("    - %v\n", v)
				}
			}
			
			// Labels
			if labels, ok := instanceData["labels"].(map[string]interface{}); ok && len(labels) > 0 {
				fmt.Println("\n  Labels:")
				for k, v := range labels {
					fmt.Printf("    %s=%v\n", k, v)
				}
			}
		}
		
		// Events
		if events, ok := result["events"].([]interface{}); ok && len(events) > 0 {
			fmt.Println("\nEvents:")
			for i, event := range events {
				if i >= 10 { // Limit to last 10 events
					fmt.Printf("  ... and %d more events\n", len(events)-10)
					break
				}
				eventStr := fmt.Sprintf("%v", event)
				// Format event timestamp if present
				if idx := strings.Index(eventStr, "] "); idx > 0 {
					tsPart := eventStr[:idx+1]
					msgPart := eventStr[idx+2:]
					// Try to parse and reformat timestamp
					tsPart = strings.Trim(tsPart, "[]")
					if t, err := time.Parse(time.RFC3339Nano, tsPart); err == nil {
						tsPart = t.Format("2006-01-02 15:04:05")
					}
					fmt.Printf("  [%s] %s\n", tsPart, msgPart)
				} else {
					fmt.Printf("  %s\n", eventStr)
				}
			}
		}
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.AddCommand(describeNodeCmd)
	describeCmd.AddCommand(describeJobCmd)
	describeCmd.AddCommand(describeInstanceCmd)
}

