package handler

import (
	"archivePNG/app/internal/model"
	"archivePNG/app/internal/service"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"strings"
	"github.com/gin-gonic/gin"
)

var (
	tasks      = make(map[string]*model.Task)
	activeTasks = 0
	CheckAct sync.Mutex
	CheckTask sync.Mutex
)

// CreateNewTask
// @Summary Создать новую задачу на архивирование
// @Description Создает новую задачу с указанным именем. Активных задач не может быть больше 3.
// @Tags tasks
// @Param task_name path string true "Имя задачи"
// @Success 200 {object} map[string]string "Сообщение об успешном создании"
// @Failure 400 {object} map[string]string "Ошибка создания (например, превышение лимита задач или задача существует)"
// @Router /tasks/{task_name} [post]
func  CreateNewTask(c  *gin.Context){
	task_name :=  c.Param("task_name")

	log.Printf("Создание новой задачи: %s", task_name)
	CheckAct.Lock()
	if activeTasks >= 3{ 
		log.Printf("Задач в данный момент 3.\n")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Сервер занят"})
		CheckAct.Unlock()
		return
	}
	CheckAct.Unlock()
	CheckTask.Lock()
	if tasks[task_name] == nil{
		tasks[task_name] = &model.Task{
			TaskName: task_name,
			Status:   model.StatusP,
			Count:    0,
		}
		CheckAct.Lock()
		activeTasks++
		CheckAct.Unlock()
	
		CheckTask.Unlock()
		c.JSON(http.StatusOK, gin.H{
			"message":   "Задача успешно создана",
			"task_name": task_name,
		})
		return
	}
	CheckTask.Unlock()
	c.JSON(http.StatusBadRequest, gin.H{"error": "Задача уже существует"})
}

// GetTasks
// @Summary Получить список всех задач
// @Description Возвращает массив всех текущих задач с их статусами и информацией
// @Tags tasks
// @Success 200 {array} model.Task
// @Router /tasks [get]
func  GetTasks(c  *gin.Context){
	CheckTask.Lock()
	
	var allTasks []*model.Task
	for _, t := range tasks {
		allTasks = append(allTasks, t)
	}
	c.JSON(http.StatusOK, allTasks)
	CheckTask.Unlock()
}

// GetTaskStatus
// @Summary Получить статус конкретной задачи
// @Description Возвращает подробную информацию по задаче, включая ссылки, статус, ошибки и ссылку на архив (если готов)
// @Tags tasks
// @Param task_name path string true "Имя задачи"
// @Success 200 {object} model.Task
// @Failure 404 {object} map[string]string "Задача не найдена"
// @Router /tasks/{task_name} [get]
func  GetTaskStatus(c  *gin.Context){
	task_name := c.Param("task_name")
	CheckTask.Lock()
	defer CheckTask.Unlock()
	task, ok := tasks[task_name]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Задача не найдена"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// AddUrl
// @Summary Добавить ссылку на файл в задачу
// @Description Добавляет ссылку на .pdf или .jpeg файл в указанную задачу. Как только добавлено 3 ссылки, задача начинает обрабатывать архив.
// @Tags tasks
// @Param task_name path string true "Имя задачи"
// @Param file_url body model.FileURLRequest true "URL файла для добавления"
// @Success 200 {object} map[string]string "Сообщение об успешном добавлении ссылки"
// @Failure 400 {object} map[string]string "Ошибка добавления (например, неверный формат, задача не существует, ссылка уже есть или задача заполнена)"
// @Router /tasks/{task_name} [put]
func AddUrl(c *gin.Context){
	task_name :=  c.Param("task_name")
	var fileUrl struct{
		Url string `json:"file_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&fileUrl); err !=nil{
		log.Printf("Ошибка парсинга тела запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	CheckTask.Lock()
	task, ok := tasks[task_name]
	CheckTask.Unlock()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Такой задачи не существует."})
		return
	}
	log.Printf("Попытка добавления в задачу '%s' ссылки файлов\n", task_name)
	if !strings.HasSuffix(fileUrl.Url, ".pdf") && !strings.HasSuffix(fileUrl.Url, ".jpeg"){
		log.Printf("Ошибка. Доступные форматы объектов .pdf и .jpeg")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Ссылка не добавлена. Не валидный формат объекта.",
			"File format": ".jpeg, .pdf",
		})
		return
	}
	task.TaskMutex.Lock()
	if tasks[task_name].Count < 3{
		if myсontains(tasks[task_name].Links, fileUrl.Url){
			c.JSON(http.StatusBadRequest, gin.H{"error": "Такая ссылка уже есть"})
			tasks[task_name].TaskMutex.Unlock()
			return	
		}
		tasks[task_name].Links = append(tasks[task_name].Links, fileUrl.Url)
		tasks[task_name].Count++
		tasks[task_name].Status = model.StatusP

		c.JSON(http.StatusOK, gin.H{
			"message":   "Ссылка успешно добавлена в задачу",
			"task_name": task_name,
			"file_url": fileUrl.Url,
		})
		if tasks[task_name].Count == 3{
			tasks[task_name].Status = model.StatusR
			tasks[task_name].TaskMutex.Unlock()
			go service.MakeArchive(task_name, tasks[task_name], &tasks[task_name].TaskMutex, &activeTasks, &CheckAct)
			return
		}
		tasks[task_name].TaskMutex.Unlock()
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Задача заполнена"})
	tasks[task_name].TaskMutex.Unlock()
}

// DeleteTask
// @Summary Удалить задачу
// @Description Удаляет задачу и освобождает слот для новых задач.
// @Tags tasks
// @Param task_name path string true "Имя задачи"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении задачи"
// @Failure 400 {object} map[string]string "Ошибка удаления (задача не найдена)"
// @Router /tasks/{task_name} [delete]
func DeleteTask(c *gin.Context){
	task_name :=  c.Param("task_name")
	log.Printf("Попытка удаления в задачи '%s'\n", task_name)
	CheckTask.Lock()
	task, ok := tasks[task_name]
	CheckTask.Unlock()
	if !ok || task == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Такой задачи не существует."})
		return
	}
	tasks[task_name].TaskMutex.Lock()
	if tasks[task_name].Status != model.StatusC{
		CheckAct.Lock()
		activeTasks--
		CheckAct.Unlock()
	}
	tasks[task_name].TaskMutex.Unlock()
	CheckTask.Lock()
	delete(tasks, task_name)
	CheckTask.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"message":   "Задача успешно удалена",
		"task_name": task_name,
	})
}

// DeleteURL
// @Summary Удалить ссылку из задачи
// @Description Удаляет ссылку по индексу из задачи, если задача не завершена.
// @Tags tasks
// @Param task_name path string true "Имя задачи"
// @Param file_url_num path int true "Порядковый номер ссылки (начиная с 1)"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении ссылки"
// @Failure 400 {object} map[string]string "Ошибка удаления (например, задача не найдена, задача завершена, неверный индекс)"
// @Router /tasks/{task_name}/{file_url_num} [delete]
func DeleteURL(c *gin.Context){
	task_name := c.Param("task_name")
	fileUrl := c.Param("file_url_num")

	log.Printf("Попытка удаления ссылки в задаче '%s'\n", task_name)
	CheckTask.Lock()
	task, ok := tasks[task_name]
	CheckTask.Unlock()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Такой задачи не существует."})
		return
	}

	tasks[task_name].TaskMutex.Lock()

	if tasks[task_name].Status == model.StatusC {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Эта задача уже завершена."})
		task.TaskMutex.Unlock()
		return
	}
	i, err := strconv.Atoi(fileUrl)
	if err != nil {
		fmt.Println("Ошибка преобразования:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Введен не порядковый номер ссылки"})
		task.TaskMutex.Unlock()
		return
	}
	if i >= 3 || i > tasks[task_name].Count{
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cсылки под таким номером в задаче нет"})
		task.TaskMutex.Unlock()
		return	
	}

	deletedLink := tasks[task_name].Links[i-1]
	tasks[task_name].Links = removeString(tasks[task_name].Links, tasks[task_name].Links[i-1])
	tasks[task_name].Count--
	tasks[task_name].TaskMutex.Lock()
	c.JSON(http.StatusOK, gin.H{
		"message":   "Cсылка успешно удалена",
		"task_name": task_name,
		"file_url_": deletedLink,
	})
}

func myсontains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func removeString(slice []string, str string) []string {
	for i, v := range slice {
		if v == str {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
