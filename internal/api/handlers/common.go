package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// isValidName checks if the name contains only valid characters
func isValidName(name string) bool {
	if name == "" {
		return false
	}
	
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || 
			(r >= 'A' && r <= 'Z') || 
			(r >= '0' && r <= '9') || 
			r == '-' || r == '_') {
			return false
		}
	}
	
	return true
}

// parsePagination parses limit and offset from query parameters
func parsePagination(c *gin.Context) (limit, offset int) {
	limit = 10
	offset = 0
	
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	
	return limit, offset
}