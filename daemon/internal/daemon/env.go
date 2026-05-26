package daemon

import (
	"bufio"
	"os"
	"strings"

	"github.com/huseinzol05/pm2-go/daemon/internal/process"
)

// buildEnv constructs the environment slice for an exec.Cmd.
func buildEnv(spec *process.Spec, instanceID int) []string {
	env := os.Environ()
	// env_files first (lowest priority among spec-provided)
	for _, f := range spec.EnvFiles {
		for k, v := range readDotEnv(f) {
			env = append(env, k+"="+v)
		}
	}
	// inline env wins
	for k, v := range spec.Env {
		env = append(env, k+"="+v)
	}
	// pm2 conventions
	env = append(env,
		"PM2_GO_INSTANCE="+itoa(instanceID),
		"PM2_GO_APP_NAME="+spec.Name,
		"NODE_APP_INSTANCE="+itoa(instanceID),
		"PM2_INSTANCE_ID="+itoa(instanceID),
	)
	return env
}

func readDotEnv(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		v = strings.Trim(v, `"'`)
		out[k] = v
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
