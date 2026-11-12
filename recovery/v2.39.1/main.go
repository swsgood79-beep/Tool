// Newly written recovery utility. This is NOT the original application's backend.
package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed evidence/assets/web evidence/asset-manifest.json evidence/report.json
var recovered embed.FS

type asset struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
	Length int    `json:"length"`
	SHA256 string `json:"sha256"`
}

func main() {
	binary := flag.String("binary", "", "verify the original EXE against recovered byte ranges")
	export := flag.String("export", "", "export embedded assets to a NEW directory (never overwrite)")
	flag.Parse()
	if err := run(*binary, *export); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(binary, export string) error {
	manifest, err := recovered.ReadFile("evidence/asset-manifest.json")
	if err != nil {
		return err
	}
	var assets []asset
	if err = json.Unmarshal(manifest, &assets); err != nil {
		return err
	}
	var original []byte
	if binary != "" {
		original, err = os.ReadFile(binary)
		if err != nil {
			return err
		}
		reportBytes, err := recovered.ReadFile("evidence/report.json")
		if err != nil {
			return err
		}
		var report struct {
			SHA256 string `json:"binary_sha256"`
		}
		if err = json.Unmarshal(reportBytes, &report); err != nil {
			return err
		}
		sum := sha256.Sum256(original)
		if hex.EncodeToString(sum[:]) != report.SHA256 {
			return fmt.Errorf("original binary SHA-256 mismatch")
		}
	}
	// Validate all bytes before creating any export directory.
	for _, a := range assets {
		if !fs.ValidPath(a.Path) || !strings.HasPrefix(a.Path, "web/") {
			return fmt.Errorf("invalid asset path %q", a.Path)
		}
		data, err := recovered.ReadFile("evidence/assets/" + a.Path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if len(data) != a.Length || hex.EncodeToString(sum[:]) != a.SHA256 {
			return fmt.Errorf("asset hash mismatch: %s", a.Path)
		}
		if binary != "" {
			if a.Offset < 0 || a.Offset > int64(len(original)) || int64(a.Length) > int64(len(original))-a.Offset {
				return fmt.Errorf("asset outside binary: %s", a.Path)
			}
			sum = sha256.Sum256(original[a.Offset : a.Offset+int64(a.Length)])
			if hex.EncodeToString(sum[:]) != a.SHA256 {
				return fmt.Errorf("binary asset mismatch: %s", a.Path)
			}
		}
	}
	if export != "" {
		if err = os.Mkdir(export, 0700); err != nil {
			return fmt.Errorf("export directory must not already exist: %w", err)
		}
		for _, a := range assets {
			data, err := recovered.ReadFile("evidence/assets/" + a.Path)
			if err != nil {
				return err
			}
			target := filepath.Join(export, filepath.FromSlash(a.Path))
			if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, writeErr := f.Write(data)
			closeErr := f.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	fmt.Printf("Verified %d extracted assets. Original backend source is not recovered.\n", len(assets))
	for _, a := range assets {
		fmt.Printf("%8d  %s\n", a.Length, a.Path)
	}
	return nil
}
