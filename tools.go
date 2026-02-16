//go:build tools
// +build tools

package tools

import (
    _ "github.com/gofiber/fiber/v2"
    _ "github.com/gofiber/fiber/v2/middleware/cors"
    _ "github.com/gofiber/fiber/v2/middleware/recover"
    _ "github.com/gofiber/fiber/v2/middleware/limiter"
    _ "golang.org/x/net/proxy"
)
