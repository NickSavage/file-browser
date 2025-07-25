package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modTime"`
	IsDir        bool      `json:"isDir"`
	Extension    string    `json:"extension"`
	RelativePath string    `json:"relativePath"`
	IsSymlink    bool      `json:"isSymlink"`
	LinkTarget   string    `json:"linkTarget,omitempty"`
}

type FileIndex struct {
	Files       []FileInfo    `json:"files"`
	Directories []FileInfo    `json:"directories"`
	LastIndexed time.Time     `json:"lastIndexed"`
	TotalFiles  int           `json:"totalFiles"`
	TotalSize   int64         `json:"totalSize"`
}

type Server struct {
	ServeDir string
	Index    *FileIndex
}

func main() {
	serveDir := os.Getenv("SERVE_DIR")
	if serveDir == "" {
		serveDir = "./data"
	}

	server := &Server{
		ServeDir: serveDir,
	}

	// Build initial index
	log.Printf("Indexing directory: %s", serveDir)
	server.buildIndex()
	log.Printf("Indexed %d files and %d directories", len(server.Index.Files), len(server.Index.Directories))

	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
	}))

	// Serve static files from React build
	r.Static("/static", "./web/build/static")
	r.StaticFile("/", "./web/build/index.html")
	r.StaticFile("/favicon.ico", "./web/build/favicon.ico")

	// API routes
	api := r.Group("/api")
	{
		api.GET("/index", server.getIndex)
		api.POST("/index/rebuild", server.rebuildIndex)
		api.GET("/browse/*path", server.browsePath)
		api.GET("/download/*path", server.downloadFile)
		api.POST("/upload/*path", server.uploadFile)
		api.PUT("/rename/*path", server.renameFile)
		api.DELETE("/delete/*path", server.deleteFile)
		api.POST("/mkdir/*path", server.createDirectory)
	}

	// Catch-all route for React Router
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/build/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}

// getFileInfo handles symlinks properly by checking if it's a symlink first,
// then getting the target info if it is
func getFileInfo(path string, name string, relativePath string) (FileInfo, error) {
	// Use Lstat to get info about the link itself (not the target)
	lstat, err := os.Lstat(path)
	if err != nil {
		return FileInfo{}, err
	}

	fileInfo := FileInfo{
		Name:         name,
		Path:         path,
		RelativePath: relativePath,
		IsSymlink:    lstat.Mode()&os.ModeSymlink != 0,
	}

	if fileInfo.IsSymlink {
		// Get the target of the symlink
		target, err := os.Readlink(path)
		if err != nil {
			// If we can't read the link, treat it as a broken symlink
			fileInfo.LinkTarget = "broken symlink"
			fileInfo.Size = 0
			fileInfo.ModTime = lstat.ModTime()
			fileInfo.IsDir = false
			fileInfo.Extension = ""
			return fileInfo, nil
		}

		fileInfo.LinkTarget = target

		// Now get info about the target
		stat, err := os.Stat(path)
		if err != nil {
			// Broken symlink - use lstat info
			fileInfo.LinkTarget = target + " (broken)"
			fileInfo.Size = 0
			fileInfo.ModTime = lstat.ModTime()
			fileInfo.IsDir = false
			fileInfo.Extension = ""
			return fileInfo, nil
		}

		// Use target's properties
		fileInfo.Size = stat.Size()
		fileInfo.ModTime = stat.ModTime()
		fileInfo.IsDir = stat.IsDir()
		if !stat.IsDir() {
			fileInfo.Extension = strings.ToLower(filepath.Ext(name))
		}
	} else {
		// Regular file or directory
		fileInfo.Size = lstat.Size()
		fileInfo.ModTime = lstat.ModTime()
		fileInfo.IsDir = lstat.IsDir()
		if !lstat.IsDir() {
			fileInfo.Extension = strings.ToLower(filepath.Ext(name))
		}
	}

	return fileInfo, nil
}

func (s *Server) buildIndex() {
	index := &FileIndex{
		Files:       make([]FileInfo, 0),
		Directories: make([]FileInfo, 0),
		LastIndexed: time.Now(),
	}

	err := filepath.Walk(s.ServeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		relativePath, _ := filepath.Rel(s.ServeDir, path)
		if relativePath == "." {
			return nil // Skip root directory
		}

		fileInfo, err := getFileInfo(path, info.Name(), relativePath)
		if err != nil {
			return nil // Skip files with errors
		}

		if fileInfo.IsDir {
			index.Directories = append(index.Directories, fileInfo)
		} else {
			index.Files = append(index.Files, fileInfo)
			index.TotalSize += fileInfo.Size
		}

		return nil
	})

	if err != nil {
		log.Printf("Error building index: %v", err)
	}

	index.TotalFiles = len(index.Files)
	s.Index = index
}

func (s *Server) getIndex(c *gin.Context) {
	c.JSON(http.StatusOK, s.Index)
}

func (s *Server) rebuildIndex(c *gin.Context) {
	s.buildIndex()
	c.JSON(http.StatusOK, gin.H{"message": "Index rebuilt successfully"})
}

func (s *Server) browsePath(c *gin.Context) {
	requestPath := c.Param("path")
	if requestPath == "" || requestPath == "/" {
		requestPath = ""
	} else {
		requestPath = strings.TrimPrefix(requestPath, "/")
	}

	fullPath := filepath.Join(s.ServeDir, requestPath)

	// Security check - ensure path is within serve directory
	if !strings.HasPrefix(fullPath, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Path not found"})
		return
	}

	if !stat.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is not a directory"})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read directory"})
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		relativePath := filepath.Join(requestPath, entry.Name())
		entryPath := filepath.Join(fullPath, entry.Name())
		
		fileInfo, err := getFileInfo(entryPath, entry.Name(), relativePath)
		if err != nil {
			continue // Skip files with errors
		}
		
		files = append(files, fileInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"path":  requestPath,
		"files": files,
	})
}

func (s *Server) downloadFile(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("path"), "/")
	fullPath := filepath.Join(s.ServeDir, requestPath)

	if !strings.HasPrefix(fullPath, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if stat.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot download directory"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fullPath)))
	c.File(fullPath)
}

func (s *Server) uploadFile(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("path"), "/")
	targetDir := filepath.Join(s.ServeDir, requestPath)

	if !strings.HasPrefix(targetDir, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	targetPath := filepath.Join(targetDir, header.Filename)
	out, err := os.Create(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Rebuild index after upload
	go s.buildIndex()

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

func (s *Server) renameFile(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("path"), "/")
	fullPath := filepath.Join(s.ServeDir, requestPath)

	if !strings.HasPrefix(fullPath, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var body struct {
		NewName string `json:"newName"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	newPath := filepath.Join(filepath.Dir(fullPath), body.NewName)
	if err := os.Rename(fullPath, newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename file"})
		return
	}

	// Rebuild index after rename
	go s.buildIndex()

	c.JSON(http.StatusOK, gin.H{"message": "File renamed successfully"})
}

func (s *Server) deleteFile(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("path"), "/")
	fullPath := filepath.Join(s.ServeDir, requestPath)

	if !strings.HasPrefix(fullPath, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	// Rebuild index after delete
	go s.buildIndex()

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

func (s *Server) createDirectory(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("path"), "/")
	
	var body struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	fullPath := filepath.Join(s.ServeDir, requestPath, body.Name)

	if !strings.HasPrefix(fullPath, s.ServeDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Rebuild index after directory creation
	go s.buildIndex()

	c.JSON(http.StatusOK, gin.H{"message": "Directory created successfully"})
}