// @title Archive ZIP API
// @version 1.0
// @description API для загрузки PDF и JPEG, архивации и получения ZIP-файла.
// @host localhost:8080
// @BasePath /
package main

import (
	"archivePNG/app/internal/handler"
	"log"
	"github.com/gin-gonic/gin"
	_ "archivePNG/app/docs"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/files"
	"os"
)


func main(){
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	port = ":" + port
	log.Printf("Запуск сервера на порту %s", port)
	
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Static("/archives", "./archives")

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.POST("/tasks/:task_name", handler.CreateNewTask)
	router.GET("/tasks", handler.GetTasks)
	router.GET("/tasks/:task_name", handler.GetTasks)
	router.PUT("/tasks/:task_name", handler.AddUrl)
	router.DELETE("/tasks/:task_name", handler.DeleteTask)
	router.DELETE("/tasks/:task_name/:file_url_num", handler.DeleteURL)
	
	log.Printf("Cервер слушает на порту %s", port)
	log.Fatal(router.Run(port))
}