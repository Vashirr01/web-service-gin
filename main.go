package main

import (
	"database/sql"
	"fmt"
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
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
	// router.GET("/hello", helloWorldHandler)
	router.GET("/", initRender)
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)
	router.DELETE("/albums/:id", deleteAlbumByID)
	router.Run("localhost:8080")
}

func render(c *gin.Context, status int, template templ.Component) error {
	c.Status(status)
	return template.Render(c.Request.Context(), c.Writer)
}

func initRender(c *gin.Context) {
	render(c, 200, test())
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
	c.IndentedJSON(http.StatusOK, albums)
}

func postAlbums(c *gin.Context) {
	var newAlbum album
	if err := c.BindJSON(&newAlbum); err != nil {
		return
	}
	insertSQL := `INSERT INTO albums (title, artist, price) VALUES (?, ?, ?);`
	var err error
	_, err = db.Exec(insertSQL, newAlbum.Title, newAlbum.Artist, newAlbum.Price)
	if err != nil {
		log.Fatal(err)
	}
	c.IndentedJSON(http.StatusCreated, newAlbum)
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
	c.IndentedJSON(http.StatusOK, gin.H{"message": "Album deleted successfully"})

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
