package helm

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRedisSentinelModeRendersRedisAuthAndTLSEnvVars(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm is not installed")
	}

	cmd := exec.Command(
		"helm",
		"template",
		"expo-open-ota",
		".",
		"--set-string",
		"cacheMode=redis-sentinel",
		"--set-string",
		"useRedisTLS=true",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helm template failed: %v\n%s", err, out)
	}

	env := deploymentEnvByName(t, out)

	for _, name := range []string{
		"REDIS_SENTINEL_ADDRS",
		"REDIS_SENTINEL_MASTER_NAME",
		"REDIS_USE_TLS",
		"REDIS_PASSWORD",
		"REDIS_USERNAME",
		"REDIS_CA_CERT_B64",
	} {
		if _, ok := env[name]; !ok {
			t.Fatalf("expected %s to be rendered in redis-sentinel mode", name)
		}
	}

	for _, name := range []string{
		"REDIS_SENTINEL_MASTER_NAME",
		"REDIS_PASSWORD",
		"REDIS_USERNAME",
		"REDIS_CA_CERT_B64",
	} {
		if !secretKeyRefOptional(env[name]) {
			t.Fatalf("expected %s to be rendered as an optional secret key ref", name)
		}
	}

	for _, name := range []string{"REDIS_HOST", "REDIS_PORT"} {
		if _, ok := env[name]; ok {
			t.Fatalf("did not expect %s to be rendered in redis-sentinel mode", name)
		}
	}
}

func deploymentEnvByName(t *testing.T, manifest []byte) map[string]map[string]any {
	t.Helper()

	decoder := yaml.NewDecoder(bytes.NewReader(manifest))
	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("failed to decode manifest: %v", err)
		}
		if doc["kind"] != "Deployment" {
			continue
		}

		containers := asSlice(t, asMap(t, asMap(t, asMap(t, doc["spec"])["template"])["spec"])["containers"])
		container := asMap(t, containers[0])
		envList := asSlice(t, container["env"])
		envByName := make(map[string]map[string]any, len(envList))
		for _, envItem := range envList {
			envMap := asMap(t, envItem)
			name, ok := envMap["name"].(string)
			if !ok {
				t.Fatalf("env item is missing string name: %#v", envMap)
			}
			envByName[name] = envMap
		}
		return envByName
	}

	t.Fatal("deployment manifest not found")
	return nil
}

func secretKeyRefOptional(env map[string]any) bool {
	valueFrom, ok := env["valueFrom"].(map[string]any)
	if !ok {
		return false
	}
	secretKeyRef, ok := valueFrom["secretKeyRef"].(map[string]any)
	if !ok {
		return false
	}
	optional, ok := secretKeyRef["optional"].(bool)
	return ok && optional
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()

	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", value)
	}
	return result
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()

	result, ok := value.([]any)
	if !ok {
		t.Fatalf("expected slice, got %T", value)
	}
	return result
}
