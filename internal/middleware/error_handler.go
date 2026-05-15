package middleware

import (
	"Novels_AI/backend/internal/pkg/common"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		// Check if any errors were added to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			c.JSON(http.StatusInternalServerError, &common.Response{
				Code: 500,
				Msg:  err.Error(),
			})
		}
	}
}
