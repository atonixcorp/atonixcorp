package compilers

import (
	"fmt"
	"strings"
)

type RuntimeConfig struct {
	CompilerName string
	Language     string
	Optimization string
	TargetGPU    bool
}

func BuildRuntime(config RuntimeConfig, service *Service) (map[string]string, error) {
	if service == nil {
		return nil, fmt.Errorf("compiler service is not configured")
	}
	compiler := service.Get(config.CompilerName)
	if compiler == nil {
		return nil, fmt.Errorf("compiler %q is not available", config.CompilerName)
	}
	language := strings.ToLower(strings.TrimSpace(config.Language))
	supported := false
	for _, candidate := range compiler.Languages {
		if language == candidate {
			supported = true
			break
		}
	}
	if !supported {
		return nil, fmt.Errorf("compiler %q does not support language %q", compiler.Name, config.Language)
	}
	if config.TargetGPU && compiler.Name != "nvhpc" {
		return nil, fmt.Errorf("compiler %q does not provide the configured GPU target", compiler.Name)
	}

	optimization := strings.TrimSpace(config.Optimization)
	switch optimization {
	case "", "O0", "O1", "O2", "O3", "Ofast":
	default:
		return nil, fmt.Errorf("unsupported optimization level %q", config.Optimization)
	}

	var executable string
	switch language {
	case "c":
		executable = "CC"
	case "c++":
		executable = "CXX"
	case "fortran":
		executable = "FC"
	}
	result := map[string]string{
		"COMPILER_NAME":      compiler.Name,
		"COMPILER_VERSION":   compiler.Version,
		"COMPILER_MODULE":    compiler.Module,
		"COMPILER_CONTAINER": compiler.Container,
		"COMPILER_LANGUAGE":  language,
	}
	if optimization != "" {
		result["COMPILER_OPTIMIZATION"] = "-" + optimization
	}
	if executable != "" {
		result[executable] = compiler.Name
	}
	if config.TargetGPU {
		result["COMPILER_TARGET_GPU"] = "true"
	}
	return result, nil
}
