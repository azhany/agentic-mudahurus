package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/tenancy"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register mounts public auth routes under the given group, plus the
// authenticated self-service routes which require the auth middleware.
func (h *Handler) Routes(public *echo.Group, authed *echo.Group) {
	public.POST("/auth/register", h.register)
	public.POST("/auth/login", h.login)
	public.POST("/auth/refresh", h.refresh)
	public.POST("/auth/logout", h.logout)
	public.POST("/auth/forgot-password", h.forgot)
	public.POST("/auth/reset-password", h.reset)
	public.POST("/auth/verify-email", h.verify)

	authed.GET("/me", h.me)
	authed.PUT("/me", h.updateProfile)
	authed.POST("/auth/change-password", h.changePassword)
}

func (h *Handler) register(c echo.Context) error {
	var in RegisterInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	t, err := h.svc.Register(c.Request().Context(), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, map[string]any{"tenant": t})
}

func (h *Handler) login(c echo.Context) error {
	var in struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	id := in.Username
	if id == "" {
		id = in.Email
	}
	pair, err := h.svc.Login(c.Request().Context(), id, in.Password)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pair)
}

func (h *Handler) refresh(c echo.Context) error {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	pair, err := h.svc.Refresh(c.Request().Context(), in.RefreshToken)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pair)
}

func (h *Handler) logout(c echo.Context) error {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.Logout(c.Request().Context(), in.RefreshToken); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) forgot(c echo.Context) error {
	var in struct {
		Email string `json:"email"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.ForgotPassword(c.Request().Context(), in.Email); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "if the email exists, a reset link has been sent"})
}

func (h *Handler) reset(c echo.Context) error {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(c.Request().Context(), in.Token, in.Password); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "password reset"})
}

func (h *Handler) verify(c echo.Context) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.VerifyEmail(c.Request().Context(), in.Token); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "email verified"})
}

func (h *Handler) me(c echo.Context) error {
	id, _ := tenancy.From(c.Request().Context())
	t, err := h.svc.Me(c.Request().Context(), id.TenantID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, t)
}

func (h *Handler) updateProfile(c echo.Context) error {
	id, _ := tenancy.From(c.Request().Context())
	var in struct {
		FullName  string `json:"full_name"`
		StoreName string `json:"store_name"`
		Phone     string `json:"phone"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	t, err := h.svc.UpdateProfile(c.Request().Context(), id.TenantID, in.FullName, in.StoreName, in.Phone)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, t)
}

func (h *Handler) changePassword(c echo.Context) error {
	id, _ := tenancy.From(c.Request().Context())
	var in struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.ChangePassword(c.Request().Context(), id.TenantID, in.OldPassword, in.NewPassword); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "password changed"})
}
