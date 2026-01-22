package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunMoleculeAttachAllowsHookedBead(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")
	beadsDir := filepath.Join(workDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("mkdir beads dir: %v", err)
	}

	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	logPath := filepath.Join(tmpDir, "bd.log")
	descPath := filepath.Join(tmpDir, "bd.desc")
	bdScript := `#!/bin/sh
set -e
if [ -n "$BD_LOG" ]; then
  echo "$*" >> "$BD_LOG"
fi

while [ "$1" != "" ]; do
  case "$1" in
    --no-daemon|--allow-stale)
      shift
      ;;
    --db)
      shift
      shift
      ;;
    *)
      break
      ;;
  esac
done

cmd="$1"
shift || true
case "$cmd" in
  show)
    issue_id="$1"
    status="${BEAD_STATUS:-hooked}"
    echo "[{\"id\":\"${issue_id}\",\"status\":\"${status}\",\"description\":\"\"}]"
    ;;
  update)
    for arg in "$@"; do
      case "$arg" in
        --description=*)
          if [ -n "$BD_DESC" ]; then
            printf "%s" "${arg#--description=}" > "$BD_DESC"
          fi
          ;;
      esac
    done
    ;;
esac
exit 0
`
	bdScriptWindows := `@echo off
setlocal enableextensions
if not "%BD_LOG%"=="" echo %*>>"%BD_LOG%"
set "cmd=%1"
if "%cmd%"=="--no-daemon" set "cmd=%2"
if "%cmd%"=="show" (
  set "issue_id=%2"
  if "%BEAD_STATUS%"=="" (
    echo [{"id":"%issue_id%","status":"hooked","description":""}]
  ) else (
    echo [{"id":"%issue_id%","status":"%BEAD_STATUS%","description":""}]
  )
  exit /b 0
)
if "%cmd%"=="update" (
  for %%A in (%*) do (
    set "arg=%%~A"
    call :writeDesc
  )
  exit /b 0
)
exit /b 0
:writeDesc
setlocal enableextensions
set "val=%arg%"
if /i not "%val:~0,14%"=="--description=" exit /b 0
set "desc=%val:~14%"
if not "%BD_DESC%"=="" (
  >"%BD_DESC%" <nul set /p =%desc%
)
exit /b 0
`
	_ = writeBDStub(t, binDir, bdScript, bdScriptWindows)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BEADS_DIR", beadsDir)
	t.Setenv("BD_LOG", logPath)
	t.Setenv("BD_DESC", descPath)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "hooked",
			status: "hooked",
		},
		{
			name:   "pinned",
			status: "pinned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BEAD_STATUS", tt.status)
			_ = os.Remove(descPath)

			if err := runMoleculeAttach(nil, []string{"gt-abc123", "mol-polecat-work"}); err != nil {
				t.Fatalf("runMoleculeAttach: %v", err)
			}

			descBytes, err := os.ReadFile(descPath)
			if err != nil {
				t.Fatalf("read description: %v", err)
			}

			desc := string(descBytes)
			if !strings.Contains(desc, "attached_molecule: mol-polecat-work") {
				t.Errorf("description missing attached_molecule:\n%s", desc)
			}

			var attachedAt string
			for _, line := range strings.Split(desc, "\n") {
				if strings.HasPrefix(line, "attached_at:") {
					attachedAt = strings.TrimSpace(strings.TrimPrefix(line, "attached_at:"))
					break
				}
			}
			if attachedAt == "" {
				t.Fatalf("description missing attached_at:\n%s", desc)
			}
			if _, err := time.Parse(time.RFC3339, attachedAt); err != nil {
				t.Errorf("attached_at not RFC3339 (%s): %v", attachedAt, err)
			}
		})
	}
}
