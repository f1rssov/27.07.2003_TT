package service
import(
	"archivePNG/app/internal/model"
	"sync"
	"log"
	"path/filepath"
	"io"
	"os"
	"archive/zip"
	"fmt"
	"net/http"
)

func MakeArchive(atchiveName string, task *model.Task, taskMutex *sync.Mutex, activeTasks *int, checkAct *sync.Mutex){
	task.TaskMutex.Lock()
	filenames := make([]string, task.Count)
	task.TaskMutex.Unlock()

	err := os.MkdirAll("archives", os.ModePerm)
	if err != nil {
		log.Printf("Не удалось создать директорию архивов: %v", err)
		return
	}
	archiveName := filepath.Join("archives", atchiveName + ".zip")

	task.Archive = fmt.Sprintf("http://localhost:8080/archives/%s.zip", atchiveName)
	
	var wg sync.WaitGroup
	errChan := make(chan error, task.Count)


	task.TaskMutex.Lock()
	for i, files := range task.Links{
		wg.Add(1)
		go func(i  int, files string){
			defer wg.Done()
			fname := filepath.Base(files)
			
			err := downloadFile(fname,files)
			if err != nil{
				errChan <- err
				return
			}
			filenames[i] = filepath.Base(files)
		}(i, files)
	}
	task.TaskMutex.Unlock()
	wg.Wait()
	close(errChan)

	for err := range errChan {
		task.Errors = append(task.Errors, err.Error())
		log.Printf("ошибка при скачивании: %v", err)
	}

	f, zipW, err := makeArchive(archiveName)
	if err != nil{
		log.Fatalf("error: %s", err)
	}
	defer f.Close()
	defer zipW.Close()

	for _, file := range filenames{
		if file == "" {
			continue
		}
		err = addFileToZip(zipW, file)
		if err != nil{
			log.Printf("error: %s", err)
		}
		defer os.Remove(file)
	}
	task.TaskMutex.Lock()
	task.Status = model.StatusC
	task.TaskMutex.Unlock()
	checkAct.Lock()
	*activeTasks -= 1
	checkAct.Unlock()
}

func downloadFile(filepath, furl string) error{
	resp, err := http.Get(furl)
	if err  != nil{
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200{
		return fmt.Errorf("файл  не скачался %s, пропуск",  furl)
	}
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func makeArchive(zname string) (*os.File, *zip.Writer, error){
	f, err := os.Create(zname)
	if err != nil{
		return f, nil, err
	}
	
	zipW := zip.NewWriter(f)
	return f, zipW, err
}

func  addFileToZip(wr  *zip.Writer,filename string) error{
	fileToZip, err := os.Open(filename)
	if err != nil{
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil{
		return err
	}
	header, err  := zip.FileInfoHeader(info)
	if err != nil{
		return err
	}
	header.Name = filepath.Base(filename)
	header.Method =  zip.Deflate
	zw, _ := wr.CreateHeader(header)
	_, err = io.Copy(zw, fileToZip)
	return err
} 