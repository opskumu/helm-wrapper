package main

import "github.com/gin-gonic/gin"

func getHelmEnvs(c *gin.Context) {
	respOK(c, settings.EnvVars())
}
