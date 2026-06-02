package catalog

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mudahurus/api/internal/httpx"
	"github.com/mudahurus/api/internal/storage"
	"github.com/mudahurus/api/internal/tenancy"
)

type Handler struct {
	svc   *Service
	store storage.Storage
}

func NewHandler(svc *Service, store storage.Storage) *Handler {
	return &Handler{svc: svc, store: store}
}

func (h *Handler) Routes(g *echo.Group) {
	g.GET("/categories", h.listCategories)
	g.POST("/categories", h.createCategory)
	g.PUT("/categories/:id", h.updateCategory)
	g.DELETE("/categories/:id", h.deleteCategory)

	g.GET("/products", h.listProducts)
	g.POST("/products", h.createProduct)
	g.GET("/products/:id", h.getProduct)
	g.PUT("/products/:id", h.updateProduct)
	g.DELETE("/products/:id", h.deleteProduct)
	g.POST("/products/:id/image", h.uploadImage)
	g.GET("/products/by-sku/:sku", h.getProductBySKU)

	// Legacy-compatible aliases (api/products, api/category) so the parity suite passes.
	g.GET("/api/products", h.listProducts)
	g.GET("/api/category", h.legacyCategories)
}

func tid(c echo.Context) uuid.UUID {
	id, _ := tenancy.From(c.Request().Context())
	return id.TenantID
}

func parseID(c echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, httpx.BadRequest("invalid id")
	}
	return id, nil
}

func (h *Handler) listProducts(c echo.Context) error {
	p := httpx.ParsePage(c)
	items, total, err := h.svc.ListProducts(c.Request().Context(), tid(c), p.Search, p.Limit, p.Offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: total, TotalRecordCount: total})
}

func (h *Handler) createProduct(c echo.Context) error {
	var in ProductInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	p, err := h.svc.CreateProduct(c.Request().Context(), tid(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, p)
}

func (h *Handler) getProduct(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	p, err := h.svc.GetProduct(c.Request().Context(), tid(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) getProductBySKU(c echo.Context) error {
	p, err := h.svc.GetProductBySKU(c.Request().Context(), tid(c), c.Param("sku"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) updateProduct(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var in ProductInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	p, err := h.svc.UpdateProduct(c.Request().Context(), tid(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) deleteProduct(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteProduct(c.Request().Context(), tid(c), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// uploadImage handles multipart product image upload (MH-202), validates type
// & size, stores it, references the key, and cleans up the old image.
func (h *Handler) uploadImage(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	tenantID := tid(c)
	file, err := c.FormFile("image")
	if err != nil {
		return httpx.BadRequest("missing 'image' file")
	}
	if file.Size > storage.MaxUploadBytes {
		return httpx.BadRequest("image exceeds maximum size")
	}
	ct := file.Header.Get("Content-Type")
	if !storage.AllowedImageMIME[ct] {
		return httpx.BadRequest("unsupported image type: " + ct)
	}
	src, err := file.Open()
	if err != nil {
		return httpx.BadRequest("cannot read upload")
	}
	defer src.Close()
	res, err := h.store.Put(c.Request().Context(), tenantID, "products", file.Filename, ct, src, file.Size)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	oldKey, err := h.svc.repo.SetProductImage(c.Request().Context(), tenantID, id, res.Key)
	if err != nil {
		_ = h.store.Delete(c.Request().Context(), res.Key)
		return mapErr(err)
	}
	if oldKey != "" && oldKey != res.Key {
		_ = h.store.Delete(c.Request().Context(), oldKey)
	}
	p, err := h.svc.GetProduct(c.Request().Context(), tenantID, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) listCategories(c echo.Context) error {
	items, err := h.svc.ListCategories(c.Request().Context(), tid(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"records": items})
}

func (h *Handler) legacyCategories(c echo.Context) error {
	items, err := h.svc.ListCategories(c.Request().Context(), tid(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, httpx.ListResponse{Records: items, QueryRecordCount: len(items), TotalRecordCount: len(items)})
}

func (h *Handler) createCategory(c echo.Context) error {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	cat, err := h.svc.CreateCategory(c.Request().Context(), tid(c), in.Name, in.Description)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, cat)
}

func (h *Handler) updateCategory(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := h.svc.UpdateCategory(c.Request().Context(), tid(c), id, in.Name, in.Description); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "updated"})
}

func (h *Handler) deleteCategory(c echo.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteCategory(c.Request().Context(), tid(c), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
