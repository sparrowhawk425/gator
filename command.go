package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sparrowhawk425/gator/internal/config"
	"github.com/sparrowhawk425/gator/internal/database"
	"github.com/sparrowhawk425/gator/internal/rss"
)

type command struct {
	Name   string
	Params []string
}

type commands struct {
	commandMap map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.commandMap[cmd.Name]
	if !ok {
		return fmt.Errorf("Unknown command: %s\n", cmd.Name)
	}
	return f(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandMap[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Params) < 1 {
		return fmt.Errorf("Command %s missing expected parameter <username>", cmd.Name)
	}
	username := cmd.Params[0]
	user, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("Error finding user %s: %v", username, err)
	}
	s.cfg.CurrentUserName = user.Name
	err = config.SetUser(*s.cfg)
	if err != nil {
		return fmt.Errorf("Error setting username for login %v", err)
	}
	fmt.Println("Successfully set username for login")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Params) < 1 {
		return fmt.Errorf("Command %s missing expected parameter <username>", cmd.Name)
	}
	username := cmd.Params[0]
	createParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	}
	user, err := s.db.CreateUser(context.Background(), createParams)
	if err != nil {
		return fmt.Errorf("Error registering user %v", err)
	}
	s.cfg.CurrentUserName = username
	err = config.SetUser(*s.cfg)
	if err != nil {
		return fmt.Errorf("Error setting username for login %v", err)
	}
	fmt.Printf("User %s successfully created: %v\n", username, user)

	return nil
}

func handlerUsers(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Error retrieving users from database: %v", err)
	}
	curUser := s.cfg.CurrentUserName
	for _, user := range users {
		if user.Name == curUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {

	if len(cmd.Params) < 1 {
		return fmt.Errorf("Command %s missing expected parameter <time_between_reqs>", cmd.Name)
	}
	timeBetweenReqs, err := time.ParseDuration(cmd.Params[0])
	if err != nil {
		return fmt.Errorf("Error parsing time duration %s: %v", cmd.Params[0], err)
	}
	fmt.Printf("Collecting feeds every %v\n", timeBetweenReqs)
	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {

	if len(cmd.Params) < 2 {
		return fmt.Errorf("Command %s missing expected parameters <name> <url>", cmd.Name)
	}
	feedName := cmd.Params[0]
	url := cmd.Params[1]
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		Url:       url,
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("Error creating feed: %v", err)
	}
	if err = addFeedFollow(s, user.ID, feed.ID); err != nil {
		return fmt.Errorf("Error creating feed follow for user %s: %v", user.Name, err)
	}
	return nil
}

func handlerFeeds(s *state, _ command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Error getting feeds from database: %v", err)
	}
	fmt.Println("Feeds:")
	for _, feed := range feeds {
		user, err := s.db.GetUserById(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("Error getting feed user: %v", err)
		}
		fmt.Println(" Feed:")
		fmt.Printf("  - Name: %s\n", feed.Name)
		fmt.Printf("  - URL: %s\n", feed.Url)
		fmt.Printf("  - User: %s\n", user.Name)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {

	if len(cmd.Params) < 1 {
		return fmt.Errorf("Command %s missing expected parameter <url>", cmd.Name)
	}
	url := cmd.Params[0]

	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("Error getting feed from database: %v", err)
	}
	if err = addFeedFollow(s, user.ID, feed.ID); err != nil {
		return fmt.Errorf("Error creating feed follow for user %s: %v", user.Name, err)
	}
	return nil
}

func handlerFollowing(s *state, _ command, user database.User) error {

	feedFollows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("Error getting feed follows for user %s: %v", user.Name, err)
	}
	fmt.Printf("User %s is following:\n", user.Name)
	for _, follow := range feedFollows {
		fmt.Printf(" - %s\n", follow.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {

	if len(cmd.Params) < 1 {
		return fmt.Errorf("Command %s missing expected parameter <url>", cmd.Name)
	}
	url := cmd.Params[0]

	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("Error getting feed from database: %v", err)
	}
	deleteParams := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	if err = s.db.DeleteFeedFollow(context.Background(), deleteParams); err != nil {
		return fmt.Errorf("Error deleting feed follow: %v", err)
	}
	fmt.Printf("User %s is no longer following %s\n", user.Name, feed.Name)
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2
	if len(cmd.Params) > 0 {
		var err error
		limit, err = strconv.Atoi(cmd.Params[0])
		if err != nil {
			return fmt.Errorf("Error converting %s to int: %v", cmd.Params[0], err)
		}
	}
	postsForUserParams := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}
	posts, err := s.db.GetPostsForUser(context.Background(), postsForUserParams)
	if err != nil {
		return fmt.Errorf("Error getting posts for user: %v", err)
	}
	fmt.Printf("Posts for user %s\n", user.Name)
	for _, post := range posts {
		fmt.Printf(" %v\n", post)
	}
	return nil
}

func handlerReset(s *state, _ command) error {
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Error reseting user table %v", err)
	}
	fmt.Println("Successfully reset users table")
	return nil
}

// Utility functions

func addFeedFollow(s *state, userId uuid.UUID, feedId uuid.UUID) error {
	feedFollowParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userId,
		FeedID:    feedId,
	}
	feedFollow, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return err
	}
	fmt.Printf("User %s is now following %s\n", feedFollow.UserName, feedFollow.FeedName)
	return nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}
	if err := s.db.MarkFeedFetched(context.Background(), feed.ID); err != nil {
		return err
	}
	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}
	fmt.Printf("Scraping feed %s:\n", rssFeed.Channel.Title)
	for _, item := range rssFeed.Channel.Item {
		addPost(s, item, feed.ID)
	}
	return nil
}

func addPost(s *state, item rss.RSSItem, feed_id uuid.UUID) error {

	validDesc := item.Description != ""
	desc := sql.NullString{
		String: item.Description,
		Valid:  validDesc,
	}
	validDate := true
	pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		fmt.Printf("Error parsing publish date %v: %v\n", item.PubDate, err)
		validDate = false
	}
	postParams := database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Title:       item.Title,
		Url:         item.Link,
		Description: desc,
		PublishedAt: sql.NullTime{
			Time:  pubDate,
			Valid: validDate,
		},
		FeedID: feed_id,
	}
	post, err := s.db.CreatePost(context.Background(), postParams)
	if err != nil {
		fmt.Printf("Error inserting post %s: %v\n", item.Title, err)
	}
	fmt.Printf("Successfully added post %s\n", post.Title)
	return nil
}

// Middleware to wrap extended functionality

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, c command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("Error getting current user from database: %v", err)
		}
		return handler(s, c, user)
	}
}
