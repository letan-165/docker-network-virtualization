package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Post struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID  string             `bson:"user_id" json:"user_id"`
	Title   string             `bson:"title" json:"title"`
	Content string             `bson:"content" json:"content"`
}

var postCollection *mongo.Collection

func main() {
	r := gin.Default()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.TODO())

	postCollection = client.Database("TTTN").Collection("posts")

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "post pong")
	})

	r.GET("/posts/:userID", getPostsByUserID)
	r.POST("/posts", createPost)
	r.DELETE("/posts/:postID", deletePost)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	r.Run(":" + port)
}

func getPostsByUserID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := c.Param("userID")

	exists, err := checkUserExists(userID)
	if err != nil {
		c.JSON(502, gin.H{"error": "cannot connect to user-service"})
		return
	}
	if !exists {
		c.JSON(404, gin.H{"error": "user does not exist"})
		return
	}

	cursor, err := postCollection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var posts []Post
	if err = cursor.All(ctx, &posts); err != nil {
		c.JSON(500, gin.H{"error": "cannot decode posts"})
		return
	}

	c.JSON(200, gin.H{
		"user_id": userID,
		"posts":   posts,
	})
}

func createPost(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var newPost Post
	if err := c.ShouldBindJSON(&newPost); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	exists, err := checkUserExists(newPost.UserID)
	if err != nil {
		c.JSON(502, gin.H{"error": "cannot connect to user-service"})
		return
	}
	if !exists {
		c.JSON(404, gin.H{"error": "user does not exist"})
		return
	}

	result, err := postCollection.InsertOne(ctx, newPost)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	newPost.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(201, newPost)
}

func deletePost(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	postID := c.Param("postID")
	objID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	res, err := postCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if res.DeletedCount == 0 {
		c.JSON(404, gin.H{"error": "post not found"})
		return
	}

	c.JSON(200, gin.H{"message": "post deleted"})
}

func checkUserExists(userID string) (bool, error) {
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8080"
	}

	url := fmt.Sprintf("%s/users/exists/%s", userServiceURL, userID)
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		Exists bool   `json:"exists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Exists, nil
}
