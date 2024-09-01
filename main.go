package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Task      string             `bson:"task"`
	Completed bool               `bson:"completed"`
}

var collection *mongo.Collection

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// MongoDB Atlas connection string
	// Replace <username> and <password> with your Atlas credentials
	connectionString := os.Getenv("MONGODB_URI")
	if connectionString == "" {
		log.Fatal("MONGODB_URI environment variable is not set")
	}

	// Set up MongoDB connection
	clientOptions := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB Atlas: %v", err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("Could not ping MongoDB Atlas: %v", err)
	}

	log.Println("Connected to MongoDB Atlas successfully")

	// Select database and collection
	database := client.Database("tododb")
	collection = database.Collection("todos")

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add-todo", addTodoHandler)
	http.HandleFunc("/toggle-todo", toggleTodoHandler)
	http.HandleFunc("/delete-todo", deleteTodoHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server is running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	todos, err := getTodos()
	if err != nil {
		log.Printf("Error getting todos: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, todos)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func addTodoHandler(w http.ResponseWriter, r *http.Request) {
	task := r.FormValue("task")
	if task != "" {
		todo := Todo{Task: task, Completed: false}
		if collection != nil {
			_, err := collection.InsertOne(context.TODO(), todo)
			if err != nil {
				log.Printf("Error inserting todo: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
	}
	indexHandler(w, r)
}

func toggleTodoHandler(w http.ResponseWriter, r *http.Request) {
	if collection == nil {
		indexHandler(w, r)
		return
	}

	id, err := primitive.ObjectIDFromHex(r.FormValue("id"))
	if err != nil {
		log.Printf("Error parsing ObjectID: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var todo Todo
	err = collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&todo)
	if err != nil {
		log.Printf("Error finding todo: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"completed": !todo.Completed}},
	)
	if err != nil {
		log.Printf("Error updating todo: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	indexHandler(w, r)
}

func deleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if collection == nil {
		indexHandler(w, r)
		return
	}

	id, err := primitive.ObjectIDFromHex(r.FormValue("id"))
	if err != nil {
		log.Printf("Error parsing ObjectID: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, err = collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		log.Printf("Error deleting todo: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	indexHandler(w, r)
}

func getTodos() ([]Todo, error) {
	var todos []Todo
	if collection == nil {
		return todos, nil
	}

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var todo Todo
		if err := cursor.Decode(&todo); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}
