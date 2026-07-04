package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gastownhall/gascity/internal/citylayout"
)

// BinaryBuildJSON reports how a running binary compares to the latest pack
// build stamp available to the status command.
type BinaryBuildJSON struct {
	Status         string `json:"status"`
	StampPath      string `json:"stamp_path,omitempty"`
	StampBinary    string `json:"stamp_binary,omitempty"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ActualSHA256   string `json:"actual_sha256,omitempty"`
	Matches        *bool  `json:"matches,omitempty"`
	BuiltAt        string `json:"built_at,omitempty"`
	Error          string `json:"error,omitempty"`
}

type beadsDoltliteBuildStamp struct {
	SchemaVersion string `json:"schema_version"`
	Pack          string `json:"pack"`
	Target        string `json:"target"`
	BuiltAt       string `json:"built_at"`
	Source        string `json:"source"`
	Output        string `json:"output"`
	InstalledTo   string `json:"installed_to"`
	BinaryPath    string `json:"binary_path"`
	SHA256        string `json:"sha256"`
	OutputSHA256  string `json:"output_sha256"`
	InstallSHA256 string `json:"install_sha256"`
	Commit        string `json:"commit"`
	Branch        string `json:"branch"`
	Version       string `json:"version"`
	Tags          string `json:"tags"`
	DoltLiteLib   string `json:"doltlite_lib"`
	GoCache       string `json:"gocache"`
	GoModCache    string `json:"gomodcache"`
	GoTmpDir      string `json:"gotmpdir"`
	GoVersion     string `json:"go_version"`
	GoFlags       string `json:"goflags"`
	CGOLdFlags    string `json:"cgo_ldflags"`
	GoVersionM    string `json:"go_version_m"`
}

func beadsDoltliteGCBuildStampPath(cityPath string) string {
	return filepath.Join(cityPath, citylayout.SystemPacksRoot, "beads-doltlite", "last-build-gc.json")
}

func processBuildCheckForCity(cityPath, binary string) *BinaryBuildJSON {
	if strings.TrimSpace(cityPath) == "" || strings.TrimSpace(binary) == "" {
		return nil
	}
	return processBuildCheckFromStamp(beadsDoltliteGCBuildStampPath(cityPath), binary)
}

func supervisorBuildCheck(binary string) *BinaryBuildJSON {
	if strings.TrimSpace(binary) == "" {
		return nil
	}
	stampPath := newestRegisteredBeadsDoltliteGCBuildStampPath()
	if stampPath == "" {
		return nil
	}
	return processBuildCheckFromStamp(stampPath, binary)
}

func newestRegisteredBeadsDoltliteGCBuildStampPath() string {
	entries, err := newSupervisorRegistry().List()
	if err != nil {
		return ""
	}
	var newestPath string
	var newestMod time.Time
	for _, entry := range entries {
		if strings.TrimSpace(entry.Path) == "" {
			continue
		}
		stampPath := beadsDoltliteGCBuildStampPath(entry.Path)
		info, err := os.Stat(stampPath)
		if err != nil || info.IsDir() {
			continue
		}
		if newestPath == "" || info.ModTime().After(newestMod) {
			newestPath = stampPath
			newestMod = info.ModTime()
		}
	}
	return newestPath
}

func processBuildCheckFromStamp(stampPath, binary string) *BinaryBuildJSON {
	data, err := os.ReadFile(stampPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return &BinaryBuildJSON{
			Status:    "stamp_unreadable",
			StampPath: stampPath,
			Error:     err.Error(),
		}
	}

	var stamp beadsDoltliteBuildStamp
	if err := json.Unmarshal(data, &stamp); err != nil {
		return &BinaryBuildJSON{
			Status:    "stamp_unreadable",
			StampPath: stampPath,
			Error:     err.Error(),
		}
	}
	expected := strings.TrimSpace(stamp.SHA256)
	if expected == "" {
		return &BinaryBuildJSON{
			Status:      "stamp_missing_sha256",
			StampPath:   stampPath,
			StampBinary: stamp.BinaryPath,
			BuiltAt:     stamp.BuiltAt,
		}
	}

	actual, err := fileSHA256(binary)
	if err != nil {
		return &BinaryBuildJSON{
			Status:         "binary_unreadable",
			StampPath:      stampPath,
			StampBinary:    stamp.BinaryPath,
			ExpectedSHA256: expected,
			BuiltAt:        stamp.BuiltAt,
			Error:          err.Error(),
		}
	}
	matches := strings.EqualFold(actual, expected)
	status := "matched"
	if !matches {
		status = "mismatch"
	}
	return &BinaryBuildJSON{
		Status:         status,
		StampPath:      stampPath,
		StampBinary:    stamp.BinaryPath,
		ExpectedSHA256: expected,
		ActualSHA256:   actual,
		Matches:        &matches,
		BuiltAt:        stamp.BuiltAt,
	}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read-only close

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func processDetailsSuffix(binary string, build *BinaryBuildJSON) string {
	var parts []string
	if binary != "" {
		parts = append(parts, "binary "+binary)
	}
	if text := binaryBuildStatusText(build); text != "" {
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return ""
	}
	return ", " + strings.Join(parts, ", ")
}

func processAuthoritySuffix(binary string, build *BinaryBuildJSON) string {
	suffix := processDetailsSuffix(binary, build)
	if suffix == "" {
		return ""
	}
	return " (" + strings.TrimPrefix(suffix, ", ") + ")"
}

func binaryBuildStatusText(build *BinaryBuildJSON) string {
	if build == nil {
		return ""
	}
	switch build.Status {
	case "matched":
		return "build matches beads-doltlite stamp"
	case "mismatch":
		return fmt.Sprintf("build mismatch beads-doltlite stamp expected %s running %s", shortSHA(build.ExpectedSHA256), shortSHA(build.ActualSHA256))
	case "stamp_unreadable":
		return "beads-doltlite build stamp unreadable"
	case "stamp_missing_sha256":
		return "beads-doltlite build stamp missing sha256"
	case "binary_unreadable":
		return "build check could not read binary"
	default:
		return strings.ReplaceAll(build.Status, "_", " ")
	}
}

func shortSHA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
