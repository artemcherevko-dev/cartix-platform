package server

import (
	"fmt"
	"gateway/internal/middleware"
	"log"
	"os"
	authpb "proto/auth"
	catalogpb "proto/catalog"
	orderpb "proto/order"
	paymentspb "proto/payments"
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

	orderConn, err := grpc.NewClient(
		os.Getenv("ORDER_SERVICE_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func(orderConn *grpc.ClientConn) {
		err := orderConn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(orderConn)
	orderClient := orderpb.NewOrderClient(orderConn)

	catalogConn, err := grpc.NewClient(
		os.Getenv("CATALOG_SERVICE_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func(catalogConn *grpc.ClientConn) {
		err := catalogConn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(catalogConn)
	catalogClient := catalogpb.NewCatalogClient(catalogConn)

	paymentsConn, err := grpc.NewClient(
		os.Getenv("PAYMENTS_SERVICE_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func(paymentsConn *grpc.ClientConn) {
		err := paymentsConn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(paymentsConn)
	paymentsClient := paymentspb.NewPaymentsClient(paymentsConn)

	handler := NewHandler(authClient, profileClient, orderClient, catalogClient, paymentsClient)

	v1 := r.Group("/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.Refresh)
			auth.GET("/validate", handler.ValidateToken)
			auth.GET("/verify", handler.VerifyEmail)
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
			orders := protected.Group("orders")
			{
				orders.POST("/", handler.CreateOrder)
				orders.GET("/", handler.ListOrders)
				orders.GET("/:id", handler.GetOrder)
			}
			cart := protected.Group("cart")
			{
				cart.GET("/", handler.GetCart)
				cart.POST("/", handler.AddToCart)
				cart.PUT("/", handler.UpdateCartItem)
				cart.DELETE("/", handler.ClearCart)
				cart.POST("/checkout", handler.CheckoutCart)
			}
			catalog := protected.Group("catalog")
			{
				catalog.GET("products", handler.ListProducts)
				catalog.GET("products/:id", handler.GetProduct)
			}
		}

		admin := v1.Group("admin")
		admin.Use(middleware.Auth())
		{
			// Orders: view — admin, manager, seller; manage — admin, manager.
			viewOrders := middleware.RequireRoles("admin", "manager", "seller")
			manageOrders := middleware.RequireRoles("admin", "manager")

			admin.GET("orders/", viewOrders, handler.AdminListOrders)
			admin.GET("orders/:id", viewOrders, handler.AdminGetOrder)
			admin.PUT("orders/:id/status", manageOrders, handler.AdminUpdateOrderStatus)
			admin.DELETE("orders/:id", manageOrders, handler.AdminDeleteOrder)

			// Payments: visibility for admin+manager; financial records are
			// deletable by admin only. Sellers cannot see payments at all.
			viewPayments := middleware.RequireRoles("admin", "manager")

			admin.GET("payments/", viewPayments, handler.AdminListPayments)
			admin.GET("payments/:id", viewPayments, handler.AdminGetPayment)
			admin.DELETE("payments/:id", middleware.Admin(), handler.AdminDeletePayment)

			// Catalog write operations stay admin-only.
			admin.POST("products/", middleware.Admin(), handler.CreateProduct)
			admin.PUT("products/:id", middleware.Admin(), handler.UpdateProduct)
			admin.PATCH("products/:id", middleware.Admin(), handler.UpdateProduct)
			admin.DELETE("products/:id", middleware.Admin(), handler.DeleteProduct)
		}
	}

	err = r.Run(fmt.Sprintf(":%s", addr))
	if err != nil {
		panic(err)
	}
}
