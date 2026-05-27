package apiclient

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

func wrapDaemonErr(err error) error {
	if err == nil {
		return nil
	}
	if daemonUnreachable(err) {
		return fmt.Errorf("daemon not running: %w", err)
	}
	return err
}

func daemonUnreachable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connectex") && strings.Contains(msg, "refused") ||
		strings.Contains(msg, "no connection could be made") {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) {
		if op.Err != nil {
			if errno, ok := op.Err.(syscall.Errno); ok && errno == syscall.ECONNREFUSED {
				return true
			}
			msg := strings.ToLower(op.Err.Error())
			if strings.Contains(msg, "refused") {
				return true
			}
		}
	}
	return false
}
