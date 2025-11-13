package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-scheduler/cli/client"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a job specification from a file",
	Long:  "Apply a job specification from a YAML file",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("f")
		if filePath == "" {
			return fmt.Errorf("file path is required. Use -f flag")
		}

		// Read YAML file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Parse YAML
		var yamlSpec map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlSpec); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Convert YAML spec to API request format
		apiReq := convertYAMLToAPIRequest(yamlSpec)

		// Submit job
		c := client.NewClient(getBaseURL())
		if err := c.LoadToken(); err != nil {
			return fmt.Errorf("failed to load token: %w", err)
		}

		result, err := c.Post("/jobs", apiReq)
		if err != nil {
			return err
		}

		// Format response nicely
		if jobID, ok := result["job_id"].(string); ok {
			fmt.Printf("✓ Job submitted successfully!\n\n")
			fmt.Println("Job ID:       ", jobID)
			if message, ok := result["message"].(string); ok {
				fmt.Println("Message:      ", message)
			}

			// Show job details if available
			if job, ok := result["job"].(map[string]interface{}); ok {
				if jobName, ok := job["job_name"].(string); ok && jobName != "" {
					fmt.Println("Job Name:     ", jobName)
				}
				if jobType, ok := job["job_type"].(string); ok && jobType != "" {
					fmt.Println("Type:         ", jobType)
				}
				if driverType, ok := job["driver_type"].(string); ok && driverType != "" {
					fmt.Println("Driver:       ", driverType)
				}
			}

			fmt.Printf("\nUse 'osctl describe job %s' to view job details.\n", jobID)
		} else {
			// Fallback to JSON if structure is unexpected
			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}
			fmt.Println("Job submitted successfully:")
			fmt.Println(string(jsonData))
		}

		return nil
	},
}

func convertYAMLToAPIRequest(yamlSpec map[string]interface{}) map[string]interface{} {
	req := make(map[string]interface{})

	// Basic fields
	if jobID, ok := yamlSpec["job_id"].(string); ok {
		req["job_id"] = jobID
	}
	if jobName, ok := yamlSpec["job_name"].(string); ok {
		req["job_name"] = jobName
	}
	if jobType, ok := yamlSpec["job_type"].(string); ok {
		req["job_type"] = jobType
	}
	if selectedClusters, ok := yamlSpec["selected_clusters"].([]interface{}); ok {
		clusters := make([]string, 0, len(selectedClusters))
		for _, c := range selectedClusters {
			if cluster, ok := c.(string); ok {
				clusters = append(clusters, cluster)
			}
		}
		req["selected_clusters"] = clusters
	}
	if driverType, ok := yamlSpec["driver_type"].(string); ok {
		req["driver"] = driverType
	}
	if workloadType, ok := yamlSpec["workload_type"].(string); ok {
		req["workload_type"] = workloadType
	}
	if command, ok := yamlSpec["command"].(string); ok {
		req["command"] = command
	}

	// Instance config
	if instanceConfig, ok := yamlSpec["instance_config"].(map[string]interface{}); ok {
		instSpec := make(map[string]interface{})
		if imageName, ok := instanceConfig["image_name"].(string); ok {
			instSpec["image"] = imageName
		}
		if entrypoint, ok := instanceConfig["entrypoint"].([]interface{}); ok {
			entrypointStrs := make([]string, 0, len(entrypoint))
			for _, e := range entrypoint {
				if ep, ok := e.(string); ok {
					entrypointStrs = append(entrypointStrs, ep)
				}
			}
			instSpec["command"] = entrypointStrs
		}
		if arguments, ok := instanceConfig["arguments"].([]interface{}); ok {
			argStrs := make([]string, 0, len(arguments))
			for _, a := range arguments {
				if arg, ok := a.(string); ok {
					argStrs = append(argStrs, arg)
				}
			}
			instSpec["args"] = argStrs
		}
		if driverOptions, ok := instanceConfig["driver_options"].(map[string]interface{}); ok {
			options := make(map[string]string)
			for k, v := range driverOptions {
				if val, ok := v.(string); ok {
					options[k] = val
				}
			}
			instSpec["options"] = options
		}
		req["instance_config"] = instSpec
	}

	// Resources
	if resources, ok := yamlSpec["resource_requirements"].(map[string]interface{}); ok {
		resSpec := make(map[string]interface{})
		if memoryLimitMB, ok := resources["memory_limit_mb"].(int64); ok {
			resSpec["memory_mb"] = memoryLimitMB
		} else if memoryLimitMB, ok := resources["memory_limit_mb"].(int); ok {
			resSpec["memory_mb"] = int64(memoryLimitMB)
		}
		if memoryReservedMB, ok := resources["memory_reserved_mb"].(int64); ok {
			resSpec["memory_reserve_mb"] = memoryReservedMB
		} else if memoryReservedMB, ok := resources["memory_reserved_mb"].(int); ok {
			resSpec["memory_reserve_mb"] = int64(memoryReservedMB)
		}
		if cpuLimitCores, ok := resources["cpu_limit_cores"].(float64); ok {
			resSpec["cpu"] = float32(cpuLimitCores)
		}
		if cpuReservedCores, ok := resources["cpu_reserved_cores"].(float64); ok {
			resSpec["cpu_reserve"] = float32(cpuReservedCores)
		}
		req["resources"] = resSpec
	}

	// Volumes
	if volumeMounts, ok := yamlSpec["volume_mounts"].([]interface{}); ok {
		volumes := make([]map[string]interface{}, 0, len(volumeMounts))
		for _, vm := range volumeMounts {
			if volMap, ok := vm.(map[string]interface{}); ok {
				vol := make(map[string]interface{})
				if sourcePath, ok := volMap["source_path"].(string); ok {
					vol["host_path"] = sourcePath
				}
				if targetPath, ok := volMap["target_path"].(string); ok {
					vol["instance_path"] = targetPath
				}
				if readOnly, ok := volMap["read_only"].(bool); ok {
					vol["read_only"] = readOnly
				}
				volumes = append(volumes, vol)
			}
		}
		req["volumes"] = volumes
	}

	// Environment variables
	if envVars, ok := yamlSpec["environment_variables"].(map[string]interface{}); ok {
		envMap := make(map[string]string)
		for k, v := range envVars {
			if val, ok := v.(string); ok {
				envMap[k] = val
			}
		}
		req["env"] = envMap
	}

	// Job metadata
	if jobMetadata, ok := yamlSpec["job_metadata"].(map[string]interface{}); ok {
		metaMap := make(map[string]string)
		for k, v := range jobMetadata {
			if val, ok := v.(string); ok {
				metaMap[k] = val
			}
		}
		req["meta"] = metaMap
	}

	return req
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringP("f", "f", "", "Path to YAML file")
	applyCmd.MarkFlagRequired("f")
}
