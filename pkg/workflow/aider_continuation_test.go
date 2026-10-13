//go:build !integration && !windows

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAiderEntrypointContinuesAfterShellResults(t *testing.T) {
	dir := t.TempDir()
	packageDir := filepath.Join(dir, "aider")
	codersDir := filepath.Join(packageDir, "coders")
	require.NoError(t, os.MkdirAll(codersDir, 0o700))
	for _, name := range []string{"__init__.py", "coders/__init__.py"} {
		require.NoError(t, os.WriteFile(filepath.Join(packageDir, name), nil, 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "io.py"), []byte(`class InputOutput:
    def tool_error(self, message="", strip=True):
        print(message)
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(codersDir, "base_coder.py"), []byte(`import os

class Coder:
    max_reflections = 3

    def show_send_output(self, completion):
        pass

    def calculate_and_show_tokens_and_cost(self, messages, completion=None):
        pass

    def run_shell_commands(self):
        if not self.shell_commands:
            return ""
        exit_code, output = run_cmd(self.shell_commands[0])
        if os.environ["AIDER_TEST_SCENARIO"] == "existing":
            self.reflected_message = "native retry"
        return output

    def run(self, with_message):
        self.num_reflections = 0
        self.cur_messages = []
        while True:
            self.reflected_message = None
            self.io.assistant_output("discovery" if self.num_reflections == 0 else "continuation")
            if self.num_reflections:
                if os.environ["AIDER_TEST_SCENARIO"] == "existing":
                    assert with_message == "native retry"
                else:
                    assert "Exit code:" in with_message
                    assert str(1 if os.environ["AIDER_TEST_SCENARIO"] == "failed" else 0) in with_message
            self.shell_commands = ["discover"] if self.num_reflections == 0 else ["safeoutputs noop"]
            if os.environ["AIDER_TEST_SCENARIO"] == "no-shell":
                self.shell_commands = []
            if os.environ["AIDER_TEST_SCENARIO"] == "loop":
                self.shell_commands = ["discover"]
            output = self.run_shell_commands()
            if output:
                self.cur_messages.extend([{"role": "user", "content": output}, {"role": "assistant", "content": "Ok"}])
            if not self.reflected_message or self.num_reflections >= self.max_reflections:
                return
            self.num_reflections += 1
            with_message = self.reflected_message
            if self.num_reflections == 1 and os.environ["AIDER_TEST_SCENARIO"] == "output":
                assert self.cur_messages[0]["content"] == "discovered packages"
            if self.num_reflections == 2:
                self.shell_commands = []
                self.reflected_message = None
                if os.environ["AIDER_TEST_SCENARIO"] != "loop":
                    return

def run_cmd(command, *args, **kwargs):
    if command == "safeoutputs noop":
        with open(os.environ["AIDER_TEST_TERMINAL"], "w") as record:
            record.write("noop")
        return 0, "safe output recorded"
    scenario = os.environ["AIDER_TEST_SCENARIO"]
    if scenario == "exception":
        raise OSError("discovery interrupted")
    return (1, "") if scenario == "failed" else (0, "discovered packages" if scenario == "output" else "")
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(packageDir, "main.py"), []byte(`import os
from .io import InputOutput
from .coders import base_coder

def main():
    coder = base_coder.Coder()
    coder.io = InputOutput()
    coder.run(with_message="perform workflow")
    assert coder.max_reflections == 3
    if os.environ["AIDER_TEST_SCENARIO"] == "loop":
        assert coder.num_reflections == 3
    return 0
`), 0o600))
	prompt := filepath.Join(dir, "prompt.md")
	require.NoError(t, os.WriteFile(prompt, []byte("perform workflow"), 0o600))
	for _, scenario := range []string{"output", "empty", "failed", "existing", "no-shell", "loop", "exception"} {
		t.Run(scenario, func(t *testing.T) {
			terminal := filepath.Join(t.TempDir(), "terminal")
			cmd := exec.Command("python3", loadAiderSample(t).Behaviors.Execution.Args...)
			cmd.Env = append(os.Environ(), "PYTHONPATH="+dir, "GH_AW_PROMPT="+prompt,
				"AIDER_TEST_SCENARIO="+scenario, "AIDER_TEST_TERMINAL="+terminal)
			output, err := cmd.CombinedOutput()
			if scenario == "exception" {
				require.Error(t, err, "%s", output)
				assert.Contains(t, string(output), "discovery interrupted")
			} else {
				require.NoError(t, err, "%s", output)
			}
			record, readErr := os.ReadFile(terminal)
			if scenario == "no-shell" || scenario == "loop" || scenario == "exception" {
				require.ErrorIs(t, readErr, os.ErrNotExist, "must not fabricate a terminal output")
			} else {
				require.NoError(t, readErr, "must consume discovery results and execute the next model command: %s", output)
				assert.Equal(t, "noop", string(record))
				assert.Contains(t, string(output), `"numTurns": 2`)
			}
		})
	}
}
