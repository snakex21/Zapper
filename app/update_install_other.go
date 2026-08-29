//go:build !windows

package main

import "errors"

func startUpdateInstaller(appDirectory, payloadRoot, updateRoot string) error {
	return errors.New("automatyczna instalacja aktualizacji jest obsługiwana tylko w Windows")
}
