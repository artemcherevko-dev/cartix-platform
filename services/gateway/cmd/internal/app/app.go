package app

import (
	"fmt"
	"gateway/cmd/internal/app/middleware"
	"log"
	"os"
	authpb "proto/auth"
	profilepb "proto/profile"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(addr string) {
	r := gin.Default()
	err := r.SetTrustedProxies([]string{"127.0.0.1"})
	if err != nil {
		return
	}

	authConn, err := grpc.NewClient(
		os.Getenv("AUTH_SERVICE_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func(authConn *grpc.ClientConn) {
		err := authConn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(authConn)
	authClient := authpb.NewAuthClient(authConn)

	profileConn, err := grpc.NewClient(
		os.Getenv("PROFILE_SERVICE_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func(profileConn *grpc.ClientConn) {
		err := profileConn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(profileConn)
	profileClient := profilepb.NewProfileClient(profileConn)

	handler := NewHandler(authClient, profileClient)

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
			profile := protected.Group("profile")
			{
				profile.GET("/", handler.GetProfile)
				profile.POST("/", handler.UpdateProfile)
			}
		}
	}

	err = r.Run(fmt.Sprintf(":%s", addr))
	if err != nil {
		panic(err)
	}
}
