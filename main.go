package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

type album struct {
	ID     string  `json: "id"`
	Title  string  `json: "title"`
	Artist string  `json: "artist"`
	Price  float64 `json: "price"`
}

func main() {
	var err error
	psqlInfo := "host=localhost port=5432 user=yourusername password=yourpassword dbname=albumsdb sslmode=disable"

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS albums(
	id SERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	artist TEXT NOT NULL,
	price DECIMAL(10,2)
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

	insertSQL := `INSERT INTO albums (title, artist, price) VALUES ($1, $2, $3) RETURNING id;`
	var id int
	err = db.QueryRow(insertSQL, newAlbum.Title, newAlbum.Artist, newAlbum.Price).Scan(&id)
	if err != nil {
		log.Fatal(err)
	}
	newAlbum.ID = fmt.Sprintf("%d", id)
	render(c, 200, Album(newAlbum))
}

func deleteAlbumByID(c *gin.Context) {
	id := c.Param("id")
	deleteSQL := `DELETE FROM albums WHERE id = $1;`
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
	err := db.QueryRow("SELECT id, title, artist, price FROM albums WHERE id = $1", id).Scan(&a.ID, &a.Title, &a.Artist, &a.Price)
	if err == sql.ErrNoRows {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
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
	var a album
	id := c.Param("id")
	updateSQL := `
        UPDATE albums
        SET title = ?, artist = ?, price = ?
        WHERE id = ?;
	`
	a.Title = c.Request.FormValue("title")
	a.Artist = c.Request.FormValue("artist")
	price := c.Request.FormValue("price")
	var err error
	a.Price, err = strconv.ParseFloat(price, 64)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid price"})
		return
	}

	_, err = db.Exec(updateSQL, a.Title, a.Artist, a.Price, id)
	if err != nil {
		log.Fatalf("Failed to update album: %v", err)
	}
	// Get the updated album to return
	a.ID = id
	render(c, 200, Album(a))
}

// func resetDatabase() error {
// 	// Delete all records
// 	_, err := db.Exec("DELETE FROM albums")
// 	if err != nil {
// 		return err
// 	}
//
// 	// Reset the autoincrement counter
// 	_, err = db.Exec("DELETE FROM sqlite_sequence WHERE name='albums'")
// 	if err != nil {
// 		return err
// 	}
//
// 	return nil
// }
