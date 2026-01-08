package utils

import "github.com/gofiber/fiber/v2"

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

// SuccessResponse sends a success response
func SuccessResponse(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(code).JSON(StandardResponse{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMetaResponse sends a success response with pagination meta
func SuccessWithMetaResponse(c *fiber.Ctx, code int, message string, data interface{}, meta interface{}) error {
	return c.Status(code).JSON(StandardResponse{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *fiber.Ctx, code int, message string) error {
	status := "error"
	if code >= 500 {
		status = "fail"
	}
	return c.Status(code).JSON(StandardResponse{
		Status:  status,
		Code:    code,
		Message: message,
	})
}

// ErrorWithDataResponse sends an error response with additional data
func ErrorWithDataResponse(c *fiber.Ctx, code int, message string, data interface{}) error {
	status := "error"
	if code >= 500 {
		status = "fail"
	}
	return c.Status(code).JSON(StandardResponse{
		Status:  status,
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// CreatePaginationMeta creates pagination metadata
func CreatePaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:        page,
		Limit:       limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}
