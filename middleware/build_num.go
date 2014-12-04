package middleware

import (
	"github.com/go-floki/floki"
)

func BuildNumberMiddleware(c *floki.Context) {
	c.Set("BuildNum", c.Floki.BuildNumber)

	// This var is intended to use in resource paths "/javascript" + DotBuildNum + ".js"
	// will not be needed when floki/jade will be patched to append build number behind the scenes
	if c.Floki.BuildNumber != "" {
		c.Set("DotBuildNum", "."+c.Floki.BuildNumber)
	} else {
		c.Set("DotBuildNum", "")
	}

	c.Next()
}
