/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package configtest

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// AddDevConfigPath adds the DevConfigDir to the viper path.
func AddDevConfigPath(v *viper.Viper) {
	devPath := GetDevConfigDir()
	if v != nil {
		v.AddConfigPath(devPath)
	} else {
		viper.AddConfigPath(devPath)
	}
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// GetDevConfigDir gets the path to the default configuration that is
// maintained with the source tree. This should only be used in a
// test/development context.
func GetDevConfigDir() string {
	var errs []error
	for _, getter := range []func() ([]string, error){
		envDevConfigDir,
		goModCacheConfigDir,
		goModDevConfigDir,
		goPathDevConfigDir,
	} {
		paths, err := getter()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, path := range paths {
			if path != "" && dirExists(path) {
				return path
			}
		}

	}
	panic(errors.Join(errs...))
}

func envDevConfigDir() ([]string, error) {
	return []string{os.Getenv("FABRIC_CFG_PATH")}, nil
}

func goModCacheConfigDir() ([]string, error) {
	modCache, err := goCMD(
		"list", "-m", "-f", "{{.Dir}}", "github.com/hyperledger/fabric-x/tools",
	)
	if err != nil {
		return nil, err
	}
	if modCache == "" {
		return nil, errors.New("not in module cache")
	}
	return []string{filepath.Join(modCache, "sampleconfig")}, nil
}

func goPathDevConfigDir() ([]string, error) {
	gopath, err := goCMD("env", "GOPATH")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range filepath.SplitList(gopath) {
		paths = append(
			paths,
			filepath.Join(p, "src/github.com/hyperledger/fabric-x/tools/sampleconfig"),
		)
	}
	return paths, nil
}

func goModDevConfigDir() ([]string, error) {
	modFile, err := goCMD("env", "GOMOD")
	if err != nil {
		return nil, err
	}
	if modFile == "" {
		return nil, errors.New("not a module or not in module mode")
	}
	return []string{filepath.Join(filepath.Dir(modFile), "sampleconfig")}, nil
}

// GetDevMspDir gets the path to the sampleconfig/msp tree that is maintained
// with the source tree.  This should only be used in a test/development
// context.
func GetDevMspDir() string {
	devDir := GetDevConfigDir()
	return filepath.Join(devDir, "msp")
}

func SetDevFabricConfigPath(t *testing.T) {
	t.Helper()
	t.Setenv("FABRIC_CFG_PATH", GetDevConfigDir())
}

func goCMD(vars ...string) (string, error) {
	cmd := exec.Command("go", vars...)
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
