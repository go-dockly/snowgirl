package onnx

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/multierr"
)

var gitURL = "https://github.com/microsoft/onnxruntime/releases/download/"
var target = "onnxruntime"
var version = "1.20.0"
var localPath = os.Getenv("HOME") + `/.local/lib`

func LibPath() string {
	dist, arch, err := determinePlatform()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/%s-%s-%s-%s/lib/lib%s.%s.%s",
		localPath, target, dist, arch, version, target, version, determineExtension(dist))
}

func GitPath() string {
	dist, arch, err := determinePlatform()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("v%s/%s-%s-%s-%s.tgz", version, target, dist, arch, version)
}

func FetchRuntime() error {
	var libPath = LibPath()
	_, err := os.Stat(libPath)
	if err == nil {
		return nil
	}
	if err = downloadFile(); err != nil {
		return fmt.Errorf("failed to download onnx runtime: %v", err)
	}
	return nil
}

func determinePlatform() (dist, arch string, err error) {
	switch runtime.GOOS {
	case "darwin":
		dist = "osx"
	case "linux":
		dist = "linux"
	default:
		return "", "", fmt.Errorf("OS '%s' is not supported", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	case "amd64":
		arch = "x64"
	default:
		return "", "", fmt.Errorf("architecture '%s' is not supported", runtime.GOARCH)
	}
	return dist, arch, nil
}

func determineExtension(dist string) string {
	switch dist {
	case "osx":
		return "dylib"
	case "linux":
		return "so"
	default:
		return ""
	}
}

func downloadFile() (err error) {
	if err = os.MkdirAll(localPath, 0755); err != nil {
		return err
	}
	var tgz = filepath.Join(localPath, version+".tgz")
	out, err := os.Create(tgz)
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Combine(err, out.Close(), os.Remove(tgz))
	}()
	resp, err := http.Get(gitURL + GitPath())
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer func() {
		err = multierr.Append(err, resp.Body.Close())
	}()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(os.Stderr, resp.Body)
		return fmt.Errorf("bad status code %d", resp.StatusCode)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write target: %v", err)
	}
	return unpackArchive(tgz, localPath)
}

func unpackArchive(tgzPath, dst string) (err error) {
	file, err := os.Open(tgzPath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", tgzPath, err)
	}
	defer func() {
		err = multierr.Append(err, file.Close())
	}()
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		err = fmt.Errorf("failed to read gzip archive: %w", err)
		return err
	}
	defer func() {
		err = multierr.Append(err, gzReader.Close())
	}()
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			err = fmt.Errorf("failed to read tar archive: %w", err)
			return err
		}
		// Determine file type
		targetPath := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir: // Directory
			if err = os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				err = fmt.Errorf("failed to create directory: %w", err)
				return err
			}
		default: // Regular file
			// Ensure the parent directory exists
			if err = os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				err = fmt.Errorf("failed to create parent directory: %w", err)
				return err
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				err = fmt.Errorf("failed to create file: %w", err)
				return err
			}
			defer func() {
				err = multierr.Append(err, outFile.Close())
			}()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				err = fmt.Errorf("failed to write file: %w", err)
				return err
			}
		}
	}
	return nil
}
