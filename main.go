package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type Todo struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Task      string             `bson:"task"`
	Completed bool               `bson:"completed"`
	UserID    primitive.ObjectID `bson:"user_id"`
}

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
}

var (
	client         *mongo.Client
	collection     *mongo.Collection
	userCollection *mongo.Collection
	store          *sessions.CookieStore
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Register primitive.ObjectID with gob
	gob.Register(primitive.ObjectID{})

	connectionString := os.Getenv("MONGODB_URI")
	if connectionString == "" {
		log.Fatal("MONGODB_URI environment variable is not set")
	}

	clientOptions := options.Client().ApplyURI(connectionString)
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB Atlas: %v", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("Could not ping MongoDB Atlas: %v", err)
	}

	log.Println("Connected to MongoDB Atlas successfully")

	database := client.Database("tododb")
	collection = database.Collection("todos")
	userCollection = database.Collection("users")

	// Initialize session store
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/register", registerHandler)
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
	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	userID, ok := session.Values["user_id"].(primitive.ObjectID)
	if !ok {
		log.Printf("User not authenticated, redirecting to login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	todos, err := getTodos(userID)
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := template.ParseFiles("login.html")
		if err != nil {
			log.Printf("Error parsing login template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			log.Printf("Error executing login template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	var user User
	err := userCollection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("User not found: %s", username)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			log.Printf("Database error when finding user: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("Invalid password for user %s", username)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	session, err := store.Get(r, "session")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = user.ID
	err = session.Save(r, w)
	if err != nil {
		log.Printf("Error saving session: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s logged in successfully", username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["user_id"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("register.html")
	if err != nil {
		log.Printf("Error parsing register template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Check if username already exists
		var existingUser User
		err := userCollection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&existingUser)
		if err == nil {
			tmpl.Execute(w, map[string]string{"Error": "Username already exists"})
			return
		} else if err != mongo.ErrNoDocuments {
			log.Printf("Database error when checking username: %v", err)
			tmpl.Execute(w, map[string]string{"Error": "An error occurred. Please try again."})
			return
		}

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			tmpl.Execute(w, map[string]string{"Error": "An error occurred. Please try again."})
			return
		}

		// Create a new user
		user := User{
			Username: username,
			Password: string(hashedPassword),
		}

		// Insert the user into the database
		_, err = userCollection.InsertOne(context.TODO(), user)
		if err != nil {
			log.Printf("Error inserting user: %v", err)
			tmpl.Execute(w, map[string]string{"Error": "An error occurred. Please try again."})
			return
		}

		// Redirect to login page after successful registration
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func addTodoHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(primitive.ObjectID)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	task := r.FormValue("task")
	if task != "" {
		todo := Todo{Task: task, Completed: false, UserID: userID}
		_, err := collection.InsertOne(context.TODO(), todo)
		if err != nil {
			log.Printf("Error inserting todo: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
	indexHandler(w, r)
}

func toggleTodoHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(primitive.ObjectID)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := primitive.ObjectIDFromHex(r.FormValue("id"))
	if err != nil {
		log.Printf("Error parsing ObjectID: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var todo Todo
	err = collection.FindOne(context.TODO(), bson.M{"_id": id, "user_id": userID}).Decode(&todo)
	if err != nil {
		log.Printf("Error finding todo: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
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
	session, _ := store.Get(r, "session")
	userID, ok := session.Values["user_id"].(primitive.ObjectID)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id, err := primitive.ObjectIDFromHex(r.FormValue("id"))
	if err != nil {
		log.Printf("Error parsing ObjectID: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, err = collection.DeleteOne(context.TODO(), bson.M{"_id": id, "user_id": userID})
	if err != nil {
		log.Printf("Error deleting todo: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	indexHandler(w, r)
}

func getTodos(userID primitive.ObjectID) ([]Todo, error) {
	var todos []Todo
	cursor, err := collection.Find(context.TODO(), bson.M{"user_id": userID})
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
