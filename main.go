package main

import (
	"database/sql"
	"fmt"
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var db *sql.DB

type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

func main() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Printf("Warning: error loading .env file")
	}

	// First connect to default postgres database
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
	)

	// Connect to postgres database first
	tempDB, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	// Create the albums database if it doesn't exist
	_, err = tempDB.Exec("CREATE DATABASE albums")
	if err != nil {
		log.Printf("Notice: %v", err) // Might error if DB exists, that's ok
	}

	tempDB.Close()

	// Now connect to the albums database
	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=albums sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
	)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Add retry logic for the connection
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database, attempt %d/%d: %v", i+1, maxRetries, err)
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		log.Fatal("Failed to connect to database after multiple attempts:", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS albums(
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        artist TEXT NOT NULL,
        price DECIMAL(10,2) NOT NULL
    )`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created successfully!")

	router := gin.Default()
	router.GET("/", getAlbums)
	router.GET("/:id", getAlbumByID)
	router.POST("/", postAlbums)
	router.PUT("/:id", updateAlbumByID)
	router.DELETE("/:id", deleteAlbumByID)

	// Changed from localhost:8080 to :8080 to listen on all interfaces
	router.Run(":8080")
}

func render(c *gin.Context, status int, template templ.Component) error {
	c.Status(status)
	return template.Render(c.Request.Context(), c.Writer)
}

func getAlbums(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, artist, price FROM albums")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var albums []album
	for rows.Next() {
		var a album
		if err := rows.Scan(&a.ID, &a.Title, &a.Artist, &a.Price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		albums = append(albums, a)
	}
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		render(c, 200, AlbumsDiv(albums))
		return
	}
	render(c, 200, MainTemp(AlbumsDiv(albums)))
}

func postAlbums(c *gin.Context) {
	if title := c.PostForm("title"); title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if artist := c.PostForm("artist"); artist == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artist is required"})
		return
	}

	var newAlbum album
	newAlbum.Title = c.PostForm("title")
	newAlbum.Artist = c.PostForm("artist")
	price := c.PostForm("price")
	var err error
	newAlbum.Price, err = strconv.ParseFloat(price, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	insertSQL := `INSERT INTO albums (title, artist, price) VALUES ($1, $2, $3) RETURNING id;`
	var id int
	err = db.QueryRow(insertSQL, newAlbum.Title, newAlbum.Artist, newAlbum.Price).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newAlbum.ID = fmt.Sprintf("%d", id)
	render(c, 200, Album(newAlbum))
}

func deleteAlbumByID(c *gin.Context) {
	id := c.Param("id")
	deleteSQL := `DELETE FROM albums WHERE id = $1;`
	res, err := db.Exec(deleteSQL, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}
	getAlbums(c)
}

func getAlbumByID(c *gin.Context) {
	var a album
	id := c.Param("id")
	err := db.QueryRow("SELECT id, title, artist, price FROM albums WHERE id = $1", id).Scan(&a.ID, &a.Title, &a.Artist, &a.Price)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	header := c.GetHeader("getReq")
	if header == "update" {
		render(c, 200, UpdateForm(a))
	}
	if header == "cancel" {
		render(c, 200, Album(a))
	}
}

func updateAlbumByID(c *gin.Context) {
	if title := c.PostForm("title"); title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if artist := c.PostForm("artist"); artist == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artist is required"})
		return
	}

	var a album
	id := c.Param("id")
	updateSQL := `
        UPDATE albums
        SET title = $1, artist = $2, price = $3 
        WHERE id = $4;
	`
	a.Title = c.Request.FormValue("title")
	a.Artist = c.Request.FormValue("artist")
	price := c.Request.FormValue("price")
	var err error
	a.Price, err = strconv.ParseFloat(price, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}

	res, err := db.Exec(updateSQL, a.Title, a.Artist, a.Price, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "album not found"})
		return
	}

	// Get the updated album to return
	a.ID = id
	render(c, 200, Album(a))
}
