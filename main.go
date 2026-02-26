package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/sparrowhawk425/gator/internal/config"
	"github.com/sparrowhawk425/gator/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

func main() {
	// Read from config file
	gatorConfig := config.Read()

	// Connect to the DB
	db, err := sql.Open("postgres", gatorConfig.DbUrl)
	if err != nil {
		fmt.Printf("Error accessing database: %v", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	// Initialize state
	programState := state{
		db:  dbQueries,
		cfg: &gatorConfig,
	}

	// Create commands map
	commands := commands{
		commandMap: map[string]func(*state, command) error{},
	}
	commands.register("login", handlerLogin)
	commands.register("register", handlerRegister)
	commands.register("users", handlerUsers)
	commands.register("agg", handlerAgg)
	commands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	commands.register("feeds", handlerFeeds)
	commands.register("follow", middlewareLoggedIn(handlerFollow))
	commands.register("following", middlewareLoggedIn(handlerFollowing))
	commands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	commands.register("browse", middlewareLoggedIn(handlerBrowse))
	commands.register("reset", handlerReset)

	// Process command line arguments
	cmdLineArgs := os.Args
	if len(cmdLineArgs) < 2 {
		fmt.Println("Usage: Missing command argument")
		os.Exit(1)
	}
	cmdName := cmdLineArgs[1]
	var params []string
	if len(cmdLineArgs) > 2 {
		params = cmdLineArgs[2:]
	}
	cmd := command{
		Name:   cmdName,
		Params: params,
	}

	err = commands.run(&programState, cmd)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
