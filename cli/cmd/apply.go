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

	// Check if this is a template.yaml format (has "services" array)
	if services, ok := yamlSpec["services"].([]interface{}); ok && len(services) > 0 {
		// Template.yaml format - take the first service
		service := services[0].(map[string]interface{})
		return convertTemplateServiceToAPIRequest(service)
	}

	// Legacy format - direct job specification
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
	if commandArray, ok := yamlSpec["command_array"].([]interface{}); ok {
		cmdArr := make([]string, 0, len(commandArray))
		for _, c := range commandArray {
			if cmd, ok := c.(string); ok {
				cmdArr = append(cmdArr, cmd)
			}
		}
		req["command_array"] = cmdArr
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

// convertTemplateServiceToAPIRequest converts a template.yaml service to API request format
func convertTemplateServiceToAPIRequest(service map[string]interface{}) map[string]interface{} {
	req := make(map[string]interface{})

	// Service name becomes job name
	if name, ok := service["name"].(string); ok {
		req["job_name"] = name
		req["job_type"] = "service" // Default for template.yaml services
	}

	// Service type maps to workload_type
	if serviceType, ok := service["type"].(string); ok {
		switch serviceType {
		case "oci-container":
			req["workload_type"] = "container"
			req["driver"] = "podman" // Default for OCI containers
		case "system-instance":
			req["workload_type"] = "vm"
			req["driver"] = "incus" // Default for system instances
		}
	}

	// Replicas
	if replicas, ok := service["replicas"].(int); ok {
		replicasInt32 := int32(replicas)
		req["replicas"] = replicasInt32
	}

	// Placement constraints
	if placement, ok := service["placement"].(map[string]interface{}); ok {
		placementReq := make(map[string]interface{})
		if constraints, ok := placement["constraints"].([]interface{}); ok {
			constraintStrs := make([]string, 0, len(constraints))
			for _, c := range constraints {
				if constraint, ok := c.(string); ok {
					constraintStrs = append(constraintStrs, constraint)
				}
			}
			placementReq["constraints"] = constraintStrs
		}
		if strategy, ok := placement["strategy"].(string); ok {
			placementReq["strategy"] = strategy
		}
		if len(placementReq) > 0 {
			req["placement"] = placementReq
		}
	}

	// Spec section
	if spec, ok := service["spec"].(map[string]interface{}); ok {
		// Image
		if image, ok := spec["image"].(string); ok {
			if req["instance_config"] == nil {
				req["instance_config"] = make(map[string]interface{})
			}
			instConfig := req["instance_config"].(map[string]interface{})
			instConfig["image"] = image
		}

		// Command (array format)
		if command, ok := spec["command"].([]interface{}); ok {
			cmdArr := make([]string, 0, len(command))
			for _, c := range command {
				if cmd, ok := c.(string); ok {
					cmdArr = append(cmdArr, cmd)
				}
			}
			req["command_array"] = cmdArr
		}

		// Working directory
		if workingDir, ok := spec["working_dir"].(string); ok {
			req["working_dir"] = workingDir
		}

		// Ports
		if ports, ok := spec["ports"].([]interface{}); ok {
			portMappings := make([]map[string]interface{}, 0, len(ports))
			for _, p := range ports {
				if portMap, ok := p.(map[string]interface{}); ok {
					pm := make(map[string]interface{})
					if hostPort, ok := portMap["host"].(int); ok {
						pm["host_port"] = int32(hostPort)
					}
					if containerPort, ok := portMap["container"].(int); ok {
						pm["container_port"] = int32(containerPort)
					}
					if protocol, ok := portMap["protocol"].(string); ok {
						pm["protocol"] = protocol
					}
					portMappings = append(portMappings, pm)
				}
			}
			if len(portMappings) > 0 {
				req["ports"] = portMappings
			}
		}

		// Environment variables
		if env, ok := spec["env"].([]interface{}); ok {
			envMap := make(map[string]string)
			for _, e := range env {
				if envEntry, ok := e.(map[string]interface{}); ok {
					if name, ok := envEntry["name"].(string); ok {
						if value, ok := envEntry["value"].(string); ok {
							envMap[name] = value
						}
					}
				}
			}
			if len(envMap) > 0 {
				req["env"] = envMap
			}
		}

		// Mounts
		if mounts, ok := spec["mounts"].([]interface{}); ok {
			volumes := make([]map[string]interface{}, 0, len(mounts))
			for _, m := range mounts {
				if mountMap, ok := m.(map[string]interface{}); ok {
					vol := make(map[string]interface{})
					if source, ok := mountMap["source"].(string); ok {
						vol["host_path"] = source
					}
					if target, ok := mountMap["target"].(string); ok {
						vol["instance_path"] = target
					}
					if readOnly, ok := mountMap["read_only"].(bool); ok {
						vol["read_only"] = readOnly
					}
					if mountType, ok := mountMap["type"].(string); ok {
						vol["type"] = mountType
					}
					volumes = append(volumes, vol)
				}
			}
			if len(volumes) > 0 {
				req["volumes"] = volumes
			}
		}

		// Resources
		if resources, ok := spec["resources"].(map[string]interface{}); ok {
			resReq := make(map[string]interface{})
			if cpuLimit, ok := resources["cpu_limit"].(string); ok {
				// Parse "0.5" format
				var cpu float32
				fmt.Sscanf(cpuLimit, "%f", &cpu)
				resReq["cpu"] = cpu
			}
			if memLimit, ok := resources["mem_limit"].(string); ok {
				// Parse "256MB" format
				var memMB int64
				if len(memLimit) > 2 && memLimit[len(memLimit)-2:] == "MB" {
					fmt.Sscanf(memLimit[:len(memLimit)-2], "%d", &memMB)
				} else if len(memLimit) > 2 && memLimit[len(memLimit)-2:] == "GB" {
					var memGB int64
					fmt.Sscanf(memLimit[:len(memLimit)-2], "%d", &memGB)
					memMB = memGB * 1024
				}
				resReq["memory_mb"] = memMB
			}
			if len(resReq) > 0 {
				req["resources"] = resReq
			}
		}

		// Security settings
		if security, ok := spec["security"].(map[string]interface{}); ok {
			secReq := make(map[string]interface{})
			if privileged, ok := security["privileged"].(bool); ok {
				secReq["privileged"] = privileged
			}
			if capabilities, ok := security["capabilities"].(map[string]interface{}); ok {
				if add, ok := capabilities["add"].([]interface{}); ok {
					addStrs := make([]string, 0, len(add))
					for _, a := range add {
						if capStr, ok := a.(string); ok {
							addStrs = append(addStrs, capStr)
						}
					}
					secReq["capabilities_add"] = addStrs
				}
				if drop, ok := capabilities["drop"].([]interface{}); ok {
					dropStrs := make([]string, 0, len(drop))
					for _, d := range drop {
						if capStr, ok := d.(string); ok {
							dropStrs = append(dropStrs, capStr)
						}
					}
					secReq["capabilities_drop"] = dropStrs
				}
			}
			if readOnlyRoot, ok := security["read_only_root_filesystem"].(bool); ok {
				secReq["read_only_root_filesystem"] = readOnlyRoot
			}
			if len(secReq) > 0 {
				req["security"] = secReq
			}
		}

		// Health check
		if healthCheck, ok := spec["health_check"].(map[string]interface{}); ok {
			hcReq := make(map[string]interface{})
			if test, ok := healthCheck["test"].([]interface{}); ok {
				testStrs := make([]string, 0, len(test))
				for _, t := range test {
					if testStr, ok := t.(string); ok {
						testStrs = append(testStrs, testStr)
					}
				}
				hcReq["test"] = testStrs
			}
			if interval, ok := healthCheck["interval"].(string); ok {
				hcReq["interval"] = interval
			}
			if timeout, ok := healthCheck["timeout"].(string); ok {
				hcReq["timeout"] = timeout
			}
			if retries, ok := healthCheck["retries"].(int); ok {
				hcReq["retries"] = int32(retries)
			}
			if startPeriod, ok := healthCheck["start_period"].(string); ok {
				hcReq["start_period"] = startPeriod
			}
			if len(hcReq) > 0 {
				req["health_check"] = hcReq
			}
		}

		// Restart policy
		if restartPolicy, ok := spec["restart_policy"].(map[string]interface{}); ok {
			rpReq := make(map[string]interface{})
			if condition, ok := restartPolicy["condition"].(string); ok {
				rpReq["condition"] = condition
			}
			if maxAttempts, ok := restartPolicy["max_attempts"].(int); ok {
				rpReq["max_attempts"] = int32(maxAttempts)
			}
			if len(rpReq) > 0 {
				req["restart_policy"] = rpReq
			}
		}

		// Networks
		if networks, ok := spec["networks"].([]interface{}); ok {
			networkStrs := make([]string, 0, len(networks))
			for _, n := range networks {
				if netStr, ok := n.(string); ok {
					networkStrs = append(networkStrs, netStr)
				}
			}
			if len(networkStrs) > 0 {
				req["networks"] = networkStrs
			}
		}

		// Instance type (for system-instance)
		if instanceType, ok := spec["instance_type"].(string); ok {
			req["instance_type"] = instanceType
		}

		// Image source (for system-instance)
		if source, ok := spec["source"].(map[string]interface{}); ok {
			if req["instance_config"] == nil {
				req["instance_config"] = make(map[string]interface{})
			}
			instConfig := req["instance_config"].(map[string]interface{})
			imgSource := make(map[string]interface{})
			if alias, ok := source["alias"].(string); ok {
				imgSource["alias"] = alias
			}
			if server, ok := source["server"].(string); ok {
				imgSource["server"] = server
			}
			if mode, ok := source["mode"].(string); ok {
				imgSource["mode"] = mode
			}
			if len(imgSource) > 0 {
				instConfig["image_source"] = imgSource
			}
		}

		// User data (for system-instance)
		if userData, ok := spec["user_data"].(string); ok {
			if req["instance_config"] == nil {
				req["instance_config"] = make(map[string]interface{})
			}
			instConfig := req["instance_config"].(map[string]interface{})
			instConfig["user_data"] = userData
		}

		// Devices (for system-instance)
		if devices, ok := spec["devices"].(map[string]interface{}); ok {
			deviceList := make([]map[string]interface{}, 0)
			for name, deviceData := range devices {
				if deviceMap, ok := deviceData.(map[string]interface{}); ok {
					dev := make(map[string]interface{})
					dev["name"] = name
					if devType, ok := deviceMap["type"].(string); ok {
						dev["type"] = devType
					}
					props := make(map[string]string)
					for k, v := range deviceMap {
						if k != "type" && k != "name" {
							if vStr, ok := v.(string); ok {
								props[k] = vStr
							}
						}
					}
					if len(props) > 0 {
						dev["properties"] = props
					}
					deviceList = append(deviceList, dev)
				}
			}
			if len(deviceList) > 0 {
				if req["instance_config"] == nil {
					req["instance_config"] = make(map[string]interface{})
				}
				instConfig := req["instance_config"].(map[string]interface{})
				instConfig["devices"] = deviceList
			}
		}
	}

	return req
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringP("f", "f", "", "Path to YAML file")
	applyCmd.MarkFlagRequired("f")
}
