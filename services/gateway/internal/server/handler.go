package server

import (
	"context"
	"net/http"
	authpb "proto/auth"
	catalogpb "proto/catalog"
	orderpb "proto/order"
	paymentspb "proto/payments"
	profilepb "proto/profile"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

type Handler struct {
	authClient     authpb.AuthClient
	profileClient  profilepb.ProfileClient
	orderClient    orderpb.OrderClient
	catalogClient  catalogpb.CatalogClient
	paymentsClient paymentspb.PaymentsClient
}

func NewHandler(
	authClient authpb.AuthClient,
	profileClient profilepb.ProfileClient,
	orderClient orderpb.OrderClient,
	catalogClient catalogpb.CatalogClient,
	paymentsClient paymentspb.PaymentsClient,
) *Handler {
	return &Handler{
		authClient:     authClient,
		profileClient:  profileClient,
		orderClient:    orderClient,
		catalogClient:  catalogClient,
		paymentsClient: paymentsClient,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		Phone       string `json:"phone"`
		FirstName   string `json:"first_name"`
		MiddleName  string `json:"middle_name"`
		LastName    string `json:"last_name"`
		DateOfBirth string `json:"date_of_birth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Register(ctx, &authpb.RegisterReq{
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)

	c.JSON(200, gin.H{"status": "registered"})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Login(ctx, &authpb.LoginReq{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)
	c.JSON(200, gin.H{"status": "logged in"})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Refresh(ctx, &authpb.RefreshReq{
		RefreshToken: refreshToken,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)
	c.JSON(200, gin.H{"status": "refreshed"})
}

func (h *Handler) ValidateToken(c *gin.Context) {
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authClient.Validate(c.Request.Context(), &authpb.ValidateTokenReq{
		AccessToken: accessToken,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(200, gin.H{"status": "validated", "user": user.User})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	res, err := h.authClient.VerifyEmail(c.Request.Context(), &authpb.VerifyEmailReq{
		Token: token,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "email verified", "verified": res.Verified})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	profile, err := h.profileClient.GetProfile(c.Request.Context(), &profilepb.GetProfileReq{UserId: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	var input struct {
		FirstName   *string `json:"first_name"`
		MiddleName  *string `json:"middle_name"`
		LastName    *string `json:"last_name"`
		DateOfBirth *string `json:"date_of_birth"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.profileClient.UpdateProfile(c.Request.Context(), &profilepb.UpdateProfileReq{
		UserId:      userID,
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		DateOfBirth: input.DateOfBirth,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": profile})
}

func (h *Handler) CreateOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	var req struct {
		Email string `json:"email"`
		Items []struct {
			ProductID string `json:"product_id"`
			Quantity  int32  `json:"quantity"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := make([]*orderpb.OrderItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, &orderpb.OrderItemInput{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	email := req.Email
	if email == "" {
		email = h.userEmail(c)
	}

	res, err := h.orderClient.CreateOrder(
		contextWithMetaData(c),
		&orderpb.CreateOrderReq{
			UserId: userID,
			Email:  email,
			Items:  items,
		},
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"order": res})
}

func (h *Handler) GetCart(c *gin.Context) {
	res, err := h.orderClient.GetCart(contextWithMetaData(c), &orderpb.GetCartReq{
		UserId: c.GetString("user_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"cart": res})
}

func (h *Handler) AddToCart(c *gin.Context) {
	var req struct {
		ProductID string `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	res, err := h.orderClient.AddToCart(contextWithMetaData(c), &orderpb.AddToCartReq{
		UserId:    c.GetString("user_id"),
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"cart": res})
}

func (h *Handler) UpdateCartItem(c *gin.Context) {
	var req struct {
		ProductID string `json:"product_id"`
		Quantity  int32  `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.orderClient.UpdateCartItem(contextWithMetaData(c), &orderpb.UpdateCartItemReq{
		UserId:    c.GetString("user_id"),
		ProductId: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"cart": res})
}

func (h *Handler) ClearCart(c *gin.Context) {
	res, err := h.orderClient.ClearCart(contextWithMetaData(c), &orderpb.ClearCartReq{
		UserId: c.GetString("user_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"cleared": res.Cleared})
}

func (h *Handler) CheckoutCart(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	_ = c.ShouldBindJSON(&req)

	email := req.Email
	if email == "" {
		email = h.userEmail(c)
	}

	res, err := h.orderClient.CheckoutCart(contextWithMetaData(c), &orderpb.CheckoutCartReq{
		UserId: c.GetString("user_id"),
		Email:  email,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"order": res})
}

func (h *Handler) GetOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	order, err := h.orderClient.GetOrder(contextWithMetaData(c), &orderpb.GetOrderReq{
		OrderId: c.Param("id"),
		UserId:  userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"order": order})
}

func (h *Handler) ListOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	res, err := h.orderClient.ListOrders(contextWithMetaData(c), &orderpb.ListOrdersReq{
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"orders": res.Orders})
}

func (h *Handler) GetProduct(c *gin.Context) {
	product, err := h.catalogClient.GetProduct(c.Request.Context(), &catalogpb.GetProductReq{
		Id: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"product": product})
}

func (h *Handler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	res, err := h.catalogClient.ListProducts(c.Request.Context(), &catalogpb.ListProductsReq{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"products": res.Products, "total": res.Total})
}

func (h *Handler) AdminListOrders(c *gin.Context) {
	res, err := h.orderClient.AdminListOrders(contextWithMetaData(c), &orderpb.AdminListOrdersReq{
		UserId: c.Query("user_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"orders": res.Orders})
}

func (h *Handler) AdminGetOrder(c *gin.Context) {
	order, err := h.orderClient.AdminGetOrder(contextWithMetaData(c), &orderpb.AdminGetOrderReq{
		OrderId: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"order": order})
}

func (h *Handler) AdminDeleteOrder(c *gin.Context) {
	res, err := h.orderClient.AdminDeleteOrder(contextWithMetaData(c), &orderpb.AdminDeleteOrderReq{
		OrderId: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": res.Deleted})
}

func (h *Handler) AdminUpdateOrderStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderClient.AdminUpdateOrderStatus(contextWithMetaData(c), &orderpb.AdminUpdateOrderStatusReq{
		OrderId: c.Param("id"),
		Status:  req.Status,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"order": order})
}

func (h *Handler) AdminListPayments(c *gin.Context) {
	res, err := h.paymentsClient.AdminListPayments(contextWithMetaData(c), &paymentspb.AdminListPaymentsReq{
		OrderId: c.Query("order_id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"payments": res.Payments})
}

func (h *Handler) AdminGetPayment(c *gin.Context) {
	payment, err := h.paymentsClient.GetPayment(c.Request.Context(), &paymentspb.GetPaymentReq{
		PaymentId: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"payment": payment})
}

func (h *Handler) AdminDeletePayment(c *gin.Context) {
	res, err := h.paymentsClient.AdminDeletePayment(contextWithMetaData(c), &paymentspb.AdminDeletePaymentReq{
		PaymentId: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": res.Deleted})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		PriceMinor  int64          `json:"price_minor"`
		Currency    string         `json:"currency"`
		Category    string         `json:"category"`
		Attributes  map[string]any `json:"attributes"`
		Active      bool           `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var attrs *structpb.Struct
	if req.Attributes != nil {
		attrs, _ = structpb.NewStruct(req.Attributes)
	}

	product, err := h.catalogClient.CreateProduct(c.Request.Context(), &catalogpb.CreateProductReq{
		Name:        req.Name,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
		Category:    req.Category,
		Attributes:  attrs,
		Active:      req.Active,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"product": product})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		PriceMinor  int64          `json:"price_minor"`
		Currency    string         `json:"currency"`
		Category    string         `json:"category"`
		Attributes  map[string]any `json:"attributes"`
		Active      bool           `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var attrs *structpb.Struct
	if req.Attributes != nil {
		attrs, _ = structpb.NewStruct(req.Attributes)
	}

	product, err := h.catalogClient.UpdateProduct(c.Request.Context(), &catalogpb.UpdateProductReq{
		Id:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
		Category:    req.Category,
		Attributes:  attrs,
		Active:      req.Active,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"product": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	res, err := h.catalogClient.DeleteProduct(c.Request.Context(), &catalogpb.DeleteProductReq{
		Id: c.Param("id"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"deleted": res.Deleted})
}

// userEmail resolves the authorized user's email from their access token so
// clients never have to send it manually.
func (h *Handler) userEmail(c *gin.Context) string {
	access, err := c.Cookie("access_token")
	if err != nil {
		return ""
	}

	res, err := h.authClient.Validate(c.Request.Context(), &authpb.ValidateTokenReq{
		AccessToken: access,
	})
	if err != nil || res.GetUser() == nil {
		return ""
	}

	return res.User.Email
}

func contextWithMetaData(c *gin.Context) context.Context {
	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()

	md := metadata.Pairs(
		"x-client-ip", clientIP,
		"x-user-agent", userAgent,
	)

	ctx := metadata.NewOutgoingContext(c.Request.Context(), md)
	return ctx
}

func setCookie(c *gin.Context, access, refresh string) {
	c.SetCookie("access_token", access, 60*60*24*7, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 60*60*24*7, "/", "", false, true)
}
