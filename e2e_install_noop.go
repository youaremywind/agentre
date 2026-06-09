//go:build !e2e

package main

import "context"

// installE2EFakes 在生产/默认构建中是空操作;e2e fake 仅在 `-tags e2e` 下编译进来。
func installE2EFakes(_ context.Context) {}
