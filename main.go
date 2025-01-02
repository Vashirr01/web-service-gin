package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type album struct {
	ID     string  `json: "id"`
	Title  string  `json: "title"`
	Artist string  `json: "artist"`
	Price  float64 `json: "price"`
}

func main() {
	//TODO error handling for database code
	var err error
	db, err = sql.Open("sqlite3", "albums.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	createTableSQL := `CREATE TABLE IF NOT EXISTS albums(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	artist TEXT NOT NULL,
	price FLOAT
	)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created successfully!")
	router := gin.Default()

	router.GET("/", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/", postAlbums)
	router.DELETE("/albums/:id", deleteAlbumByID)
	router.Run("localhost:8080")
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
		rows.Scan(&a.ID, &a.Title, &a.Artist, &a.Price)
		albums = append(albums, a)
	}
	if c.GetHeader("HX-Request") == "true" {
		render(c, 200, AlbumsDiv(albums))
		return
	}
	render(c, 200, MainTemp(AlbumsDiv(albums)))
}

func postAlbums(c *gin.Context) {
	var newAlbum album
	newAlbum.Title = c.PostForm("title")
	newAlbum.Artist = c.PostForm("artist")
	price := c.PostForm("price")
	var err error
	newAlbum.Price, err = strconv.ParseFloat(price, 64)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid price"})
		return
	}
	insertSQL := `INSERT INTO albums (title, artist, price) VALUES (?, ?, ?);`
	res, err := db.Exec(insertSQL, newAlbum.Title, newAlbum.Artist, newAlbum.Price)
	if err != nil {
		log.Fatal(err)
	}
	id, err := res.LastInsertId()
	newAlbum.ID = fmt.Sprintf("%d", id)
	render(c, 200, Album(newAlbum))
}

func deleteAlbumByID(c *gin.Context) {
	id := c.Param("id")
	deleteSQL := `DELETE FROM albums WHERE id = ?;`
	res, _ := db.Exec(deleteSQL, id)
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}
	getAlbums(c)
}

func getAlbumByID(c *gin.Context) {
	var a album
	id := c.Param("id")
	err := db.QueryRow("SELECT id, title, artist, price FROM albums WHERE id = ?", id).Scan(&a.ID, &a.Title, &a.Artist, &a.Price)
	if err == sql.ErrNoRows {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}
	c.IndentedJSON(http.StatusOK, a)
}
