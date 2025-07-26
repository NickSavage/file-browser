package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	IsAdmin   bool      `json:"isAdmin" gorm:"default:false"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	IsAdmin  bool   `json:"isAdmin"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
	jwt.RegisteredClaims
}

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
	DB       *gorm.DB
	JWTKey   []byte
}

func main() {
	serveDir := os.Getenv("SERVE_DIR")
	if serveDir == "" {
		serveDir = "./data"
	}

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "filebrowser.db"
	}
	
	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." {
		err := os.MkdirAll(dbDir, 0755)
		if err != nil {
			log.Fatal("Failed to create database directory:", err)
		}
	}
	
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// JWT secret key
	jwtKey := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtKey) == 0 {
		jwtKey = []byte("your-secret-key-change-in-production")
		log.Println("Warning: Using default JWT secret. Set JWT_SECRET environment variable in production.")
	}

	server := &Server{
		ServeDir: serveDir,
		DB:       db,
		JWTKey:   jwtKey,
	}

	// Create default admin user if none exists
	server.createDefaultAdmin()

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
		// Auth routes (no authentication required)
		api.POST("/login", server.login)
		
		// Protected routes (authentication required)
		protected := api.Group("/")
		protected.Use(server.authMiddleware())
		{
			protected.GET("/index", server.getIndex)
			protected.POST("/index/rebuild", server.rebuildIndex)
			protected.GET("/browse/*path", server.browsePath)
			protected.GET("/download/*path", server.downloadFile)
			protected.POST("/upload/*path", server.uploadFile)
			protected.PUT("/rename/*path", server.renameFile)
			protected.DELETE("/delete/*path", server.deleteFile)
			protected.POST("/mkdir/*path", server.createDirectory)
			
			// User management routes (admin only)
			protected.POST("/users", server.requireAdmin(), server.createUser)
			protected.GET("/users", server.requireAdmin(), server.getUsers)
			protected.DELETE("/users/:id", server.requireAdmin(), server.deleteUser)
			protected.GET("/me", server.getCurrentUser)
			protected.PUT("/me/password", server.changePassword)
		}
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

// Authentication utility functions
func (s *Server) createDefaultAdmin() {
	var count int64
	s.DB.Model(&User{}).Count(&count)
	
	if count == 0 {
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin123"
			log.Println("Warning: Using default admin password 'admin123'. Set ADMIN_PASSWORD environment variable.")
		}
		
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash admin password:", err)
		}
		
		admin := User{
			Username: "admin",
			Password: string(hashedPassword),
			IsAdmin:  true,
		}
		
		result := s.DB.Create(&admin)
		if result.Error != nil {
			log.Fatal("Failed to create admin user:", result.Error)
		}
		
		log.Println("Created default admin user (username: admin)")
	}
}

func (s *Server) generateToken(user *User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.JWTKey)
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}
		
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return s.JWTKey, nil
		})
		
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		
		c.Set("user", claims)
		c.Next()
	}
}

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}
		
		claims, ok := user.(*Claims)
		if !ok || !claims.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// Authentication handlers
func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var user User
	result := s.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	
	token, err := s.generateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"isAdmin":  user.IsAdmin,
		},
	})
}

func (s *Server) getCurrentUser(c *gin.Context) {
	user, _ := c.Get("user")
	claims := user.(*Claims)
	
	c.JSON(http.StatusOK, gin.H{
		"id":       claims.UserID,
		"username": claims.Username,
		"isAdmin":  claims.IsAdmin,
	})
}

func (s *Server) createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	
	user := User{
		Username: req.Username,
		Password: string(hashedPassword),
		IsAdmin:  req.IsAdmin,
	}
	
	result := s.DB.Create(&user)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		}
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"isAdmin":  user.IsAdmin,
		"createdAt": user.CreatedAt,
	})
}

func (s *Server) getUsers(c *gin.Context) {
	var users []User
	result := s.DB.Find(&users)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	
	c.JSON(http.StatusOK, users)
}

func (s *Server) deleteUser(c *gin.Context) {
	userID := c.Param("id")
	id, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	
	// Prevent deleting the last admin user
	var adminCount int64
	s.DB.Model(&User{}).Where("is_admin = ?", true).Count(&adminCount)
	
	var user User
	result := s.DB.First(&user, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	if user.IsAdmin && adminCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the last admin user"})
		return
	}
	
	result = s.DB.Delete(&user)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (s *Server) changePassword(c *gin.Context) {
	user, _ := c.Get("user")
	claims := user.(*Claims)
	
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get current user from database
	var currentUser User
	result := s.DB.First(&currentUser, claims.UserID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	// Verify current password
	err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(req.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}
	
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}
	
	// Update password in database
	result = s.DB.Model(&currentUser).Update("password", string(hashedPassword))
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}