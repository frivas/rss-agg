package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/frivas/rss-agg/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type blogatorAPIConfig struct {
	DB *database.Queries
}

func main() {
	feed, err := urlToFeed("https://wagslane.dev/index.xml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(feed)
	godotenv.Load(".env")

	port := os.Getenv("PORT")
	dbURL := os.Getenv("DB_CONN_STRING")

	conn, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal(err)
	}

	db := database.New(conn)
	blogatorAPICfg := blogatorAPIConfig{
		DB: db,
	}

	go startScraping(db, 10, time.Minute)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Mount("/v1", router)
	router.Get("/readiness", readinessHandler)
	router.Get("/err", errHandler)
	router.Post("/users", blogatorAPICfg.createUser)
	router.Get("/users", blogatorAPICfg.middlewareAuth(blogatorAPICfg.getUserByAPIKey))
	router.Post("/feeds", blogatorAPICfg.middlewareAuth(blogatorAPICfg.createFeed))
	router.Post("/feeds_follows", blogatorAPICfg.middlewareAuth(blogatorAPICfg.createFeedFollows))
	router.Get("/feeds", blogatorAPICfg.getAllFeeds)
	router.Get("/feeds_follows", blogatorAPICfg.middlewareAuth(blogatorAPICfg.getAllFeedFollows))
	router.Delete("/feeds_follows/{feedFollowID}", blogatorAPICfg.middlewareAuth(blogatorAPICfg.deleteFeedFollows))
	router.Delete("/feeds/{feedID}", blogatorAPICfg.middlewareAuth(blogatorAPICfg.deleteFeed))
	router.Get("/posts", blogatorAPICfg.middlewareAuth(blogatorAPICfg.getPostsForUser))

	corsMux := MiddlewareCors(router)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	log.Printf("Serving on port %s", port)
	log.Fatal(srv.ListenAndServe())
}

func readinessHandler(w http.ResponseWriter, _ *http.Request) {
	type readiness struct {
		Status string `json:"status"`
	}
	JsonResponse(w, http.StatusOK, readiness{
		Status: "ok",
	})
}

func errHandler(w http.ResponseWriter, _ *http.Request) {
	JsonResponseError(w, http.StatusInternalServerError, "Internal Server Error")
}

func (cfg *blogatorAPIConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type userPayload struct {
		Name string `json:"name"`
	}

	type validResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Name      string    `json:"name"`
		APIKey    string    `json:"api_key"`
	}

	decoder := json.NewDecoder(r.Body)
	user := userPayload{}

	err := decoder.Decode(&user)
	if err != nil {
		JsonResponseError(w, http.StatusInternalServerError, "Couldn't decode user info")
		return
	}

	if r.Method == http.MethodPost {
		dbUser, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Name:      user.Name,
		})

		if err != nil {
			fmt.Println(err)
			JsonResponseError(w, http.StatusInternalServerError, "Couldn't create the user")
			return
		}
		JsonResponse(w, http.StatusCreated, databaseUserToUser(dbUser))
	}
}

func (cfg *blogatorAPIConfig) getUserByAPIKey(w http.ResponseWriter, r *http.Request, dbUser database.User) {
	fmt.Println("User =>", dbUser)
	JsonResponse(w, http.StatusOK, databaseUserToUser(dbUser))
}

func (cfg *blogatorAPIConfig) createFeed(w http.ResponseWriter, r *http.Request, user database.User) {
	type feedPayload struct {
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	type validResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Name      string    `json:"name"`
		APIKey    string    `json:"api_key"`
	}

	decoder := json.NewDecoder(r.Body)
	feed := feedPayload{}

	err := decoder.Decode(&feed)
	if err != nil {
		JsonResponseError(w, http.StatusInternalServerError, "Couldn't decode feed info")
		return
	}

	if r.Method == http.MethodPost {
		feedId := uuid.New()
		feed, err := cfg.DB.CreateFeed(r.Context(), database.CreateFeedParams{
			ID:        feedId,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Name:      feed.Name,
			Url:       feed.Url,
			UserID:    user.ID,
		})

		if err != nil {
			fmt.Println(err)
			JsonResponseError(w, http.StatusInternalServerError, "Couldn't create the user")
			return
		}
		feedFollow, err := cfg.DB.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			FeedID:    feedId,
			UserID:    user.ID,
		})

		if err != nil {
			fmt.Println(err)
			JsonResponseError(w, http.StatusInternalServerError, fmt.Sprintf("Couldn't create the feed follow: %s", err))
			return
		}
		JsonResponse(w, http.StatusCreated, databaseNewFeedWithFollowstoNewFeedWithFollows(feed, feedFollow))
	}
}

func (cfg *blogatorAPIConfig) getAllFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := cfg.DB.GetFeeds(r.Context())
	if err != nil {
		JsonResponseError(w, http.StatusNotFound, fmt.Sprintf("Couldn't get feeds: %v", err))
	}

	JsonResponse(w, http.StatusCreated, databaseFeedstoFeeds(feeds))
}

func (cfg *blogatorAPIConfig) createFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	type feedPayload struct {
		FeedID uuid.UUID `json:"feed_id"`
	}

	type validResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		FeedID    uuid.UUID `json:"feed_id"`
		UserID    uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	feed := feedPayload{}

	err := decoder.Decode(&feed)
	if err != nil {
		JsonResponseError(w, http.StatusInternalServerError, "Couldn't decode feed info")
		return
	}

	if r.Method == http.MethodPost {
		feed, err := cfg.DB.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			FeedID:    feed.FeedID,
			UserID:    user.ID,
		})

		if err != nil {
			fmt.Println(err)
			JsonResponseError(w, http.StatusInternalServerError, fmt.Sprintf("Couldn't create the feed follow: %s", err))
			return
		}
		JsonResponse(w, http.StatusCreated, databaseFeedFollowtoFeedFollow(feed))
	}
}

func (cfg *blogatorAPIConfig) getAllFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	feeds, err := cfg.DB.GetFeedFollows(r.Context(), user.ID)
	if err != nil {
		JsonResponseError(w, http.StatusNotFound, fmt.Sprintf("Couldn't get feeds for user: %v", err))
	}

	JsonResponse(w, http.StatusCreated, databaseFeedFollowstoFeedFollows(feeds))
}

func (cfg *blogatorAPIConfig) deleteFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	feedFollowIDStr := chi.URLParam(r, "feedFollowID")
	feedFollowID, err := uuid.Parse(feedFollowIDStr)
	if err != nil {
		JsonResponseError(w, http.StatusBadRequest, fmt.Sprintf("Couldn't parse feed follow id: %v", err))
		return
	}
	err = cfg.DB.DeleteFeedFollow(r.Context(), database.DeleteFeedFollowParams{
		ID:     feedFollowID,
		UserID: user.ID,
	})
	if err != nil {
		JsonResponseError(w, http.StatusBadRequest, fmt.Sprintf("Couldn't delete feed follow: %v", err))
	}
	JsonResponse(w, http.StatusOK, struct{}{})
}

func (cfg *blogatorAPIConfig) deleteFeed(w http.ResponseWriter, r *http.Request, user database.User) {
	feedIDStr := chi.URLParam(r, "feedID")
	feedID, err := uuid.Parse(feedIDStr)
	if err != nil {
		JsonResponseError(w, http.StatusBadRequest, fmt.Sprintf("Couldn't parse feed id: %v", err))
		return
	}
	err = cfg.DB.DeleteFeed(r.Context(), feedID)
	if err != nil {
		JsonResponseError(w, http.StatusBadRequest, fmt.Sprintf("Couldn't delete feed follow: %v", err))
	}
	JsonResponse(w, http.StatusOK, struct{}{})
}

func (cfg *blogatorAPIConfig) getPostsForUser(w http.ResponseWriter, r *http.Request, user database.User) {
	posts, err := cfg.DB.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  10,
	})
	if err != nil {
		JsonResponseError(w, http.StatusBadRequest, fmt.Sprintf("Couldn't get posts: %v", err))
		return
	}
	JsonResponse(w, http.StatusOK, databasePostsToPosts(posts))
}
