package app

import (
	"fmt"
	"gateway/cmd/internal/app/middleware"
	"log"
	authpb "proto/auth"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(addr string) {
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	authConn, err := grpc.NewClient(
		"localhost:3001",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer authConn.Close()
	authClient := authpb.NewAuthClient(authConn)

	handler := NewHandler(authClient)

	v1 := r.Group("/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.Refresh)
			auth.GET("/validate", handler.ValidateToken)
		}
		protected := v1.Group("")
		protected.Use(middleware.Auth())

		{
			protected.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Hello"})
			})
		}
	}

	err = r.Run(fmt.Sprintf(":%s", addr))
	if err != nil {
		panic(err)
	}
}
