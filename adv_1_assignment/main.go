package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	usersCollection   *mongo.Collection
	recipesCollection *mongo.Collection
)

type User struct {
	ID              int    `bson:"id"`
	Name            string `bson:"name"`
	Email           string `bson:"email"`
	FavoriteCuisine string `bson:"favorite_cuisine"`
}

type Recipe struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	Ingredients []string           `bson:"ingredients"`
	CreatedBy   primitive.ObjectID `bson:"created_by"`
}

func getNextID(collection *mongo.Collection) (int, error) {
	filter := bson.M{"_id": "user_id"}
	update := bson.M{"$inc": bson.M{"sequence_value": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result struct {
		SequenceValue int `bson:"sequence_value"`
	}
	err := collection.FindOneAndUpdate(context.Background(), filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.SequenceValue, nil
}

func main() {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	log.Println("Connected to MongoDB")

	usersCollection = client.Database("cooking_db").Collection("users")
	recipesCollection = client.Database("cooking_db").Collection("recipes")

	// Serve static files (e.g., index.html)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/recipes", handleRecipes)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id == "" {
			getAllUsers(w)
		} else {
			getUserByID(w, id)
		}
	case http.MethodPost:
		createUser(w, r)
	case http.MethodPut:
		updateUserByID(w, r)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		deleteUserByID(w, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRecipes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id == "" {
			getAllRecipes(w)
		} else {
			getRecipeByID(w, id)
		}
	case http.MethodPost:
		createRecipe(w, r)
	case http.MethodPut:
		updateRecipeByID(w, r)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		deleteRecipeByID(w, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getAllUsers(w http.ResponseWriter) {
	cursor, err := usersCollection.Find(context.Background(), bson.D{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var users []User
	for cursor.Next(context.Background()) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUserByID(w http.ResponseWriter, id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var user User
	err = usersCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newID, err := getNextID(usersCollection)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = newID
	_, err = usersCollection.InsertOne(context.Background(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
func updateUserByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var userUpdates bson.M
	if err := json.NewDecoder(r.Body).Decode(&userUpdates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	update := bson.M{"$set": userUpdates}
	_, err = usersCollection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteUserByID(w http.ResponseWriter, id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = usersCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getAllRecipes(w http.ResponseWriter) {
	cursor, err := recipesCollection.Find(context.Background(), bson.D{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	var recipes []Recipe
	for cursor.Next(context.Background()) {
		var recipe Recipe
		if err := cursor.Decode(&recipe); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		recipes = append(recipes, recipe)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

func getRecipeByID(w http.ResponseWriter, id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var recipe Recipe
	err = recipesCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&recipe)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func createRecipe(w http.ResponseWriter, r *http.Request) {
	var recipe Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	recipe.ID = primitive.NewObjectID()
	_, err := recipesCollection.InsertOne(context.Background(), recipe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func updateRecipeByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var recipeUpdates bson.M
	if err := json.NewDecoder(r.Body).Decode(&recipeUpdates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	update := bson.M{"$set": recipeUpdates}
	_, err = recipesCollection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteRecipeByID(w http.ResponseWriter, id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = recipesCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
