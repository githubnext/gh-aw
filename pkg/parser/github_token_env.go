package parser

import (
	"github.com/github/gh-aw/pkg/envutil"
	"github.com/github/gh-aw/pkg/logger"
)

func githubTokenFromEnv(log *logger.Logger) string {
	if token := envutil.GetStringFromEnv("GITHUB_TOKEN", "", log); token != "" {
		return token
	}
	return envutil.GetStringFromEnv("GH_TOKEN", "", log)
}
