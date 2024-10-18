package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moesif/moesifgin"
)

func main() {
	r := gin.New()

	moesifOptions := map[string]interface{}{
		"Application_Id":   "eyJhcHAiOiIxOTg6NzQ0IiwidmVyIjoiMi4xIiwib3JnIjoiNjQwOjEyOCIsImlhdCI6MTcyNzc0MDgwMH0.ckqMfkn4o0zgfLokgP_WvMIXAOYm77PV6ACbMAlzYC0",
		"Log_Body":         true,
		"Identify_User":    func(c *gin.Context) string { return c.Request.Header.Get("X-User-Id") },
		"Identify_Company": func(c *gin.Context) string { return c.Request.Header.Get("X-Company-Id") },
		"Get_Metadata":     func(c *gin.Context) map[string]interface{} { return map[string]interface{}{"example": "metadata"} },
	}
	r.Use(moesifgin.MoesifMiddleware(moesifOptions))

	r.Use(ExampleOtherMiddleware()) // Example to show the usage of other middleware

	r.POST("/test", func(c *gin.Context) {
		log.Println("Executing test endpoint")
		c.JSON(201, gin.H{"message": "hello world"})
		log.Println("Test endpoint executed successfully")
	})

	// Listen and serve on 0.0.0.0:8080
	r.Run(":8080")
}

// ExampleOtherMiddleware is just used to show the successful integration with other middleware
// It is not required for Moesif to work.
func ExampleOtherMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		log.Println("example middleware before request")

		c.Next()

		latency := time.Since(t)
		log.Print(latency)

		status := c.Writer.Status()
		log.Println("example middleware writing status: ", status)
	}
}
