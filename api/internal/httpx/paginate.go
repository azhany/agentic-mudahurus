package httpx

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// Page captures pagination + search query params shared by list endpoints,
// mirroring the legacy Dynatable API contract (perPage/offset/queries[search]).
type Page struct {
	Limit  int
	Offset int
	Search string
}

// ParsePage reads limit/offset/search from the query string with safe defaults.
func ParsePage(c echo.Context) Page {
	p := Page{Limit: 20, Offset: 0}
	if v := c.QueryParam("perPage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Limit = n
		}
	}
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Offset = n
		}
	}
	p.Search = c.QueryParam("queries[search]")
	if p.Search == "" {
		p.Search = c.QueryParam("search")
	}
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// ListResponse is the standard paginated list envelope (compatible with the
// legacy Dynatable shape: records + counts).
type ListResponse struct {
	Records          interface{} `json:"records"`
	QueryRecordCount int         `json:"queryRecordCount"`
	TotalRecordCount int         `json:"totalRecordCount"`
}
