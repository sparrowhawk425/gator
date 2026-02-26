# Gator Blog Aggregator

A simple CLI for aggregating blogs from RSS feeds and storing them to a local Postgres Database. The program uses Go to take commands to add users and feeds to a Postgres database and then allows the user to aggregate the feeds at time intervals. It uses sqlc to generate the database integration layer.

## Installing

The gator program can be installed by running: `go install` and then exporting the install directory to your PATH.

## Set Up

The gator application uses a config file stored in the home directory, ~/.gatorconfig.json, which stores the database URL and the current user. It is formatted like this:

```
{
    "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
    "current_user_name": "my_user_name"
}
```
## Commands

The program provides several commands that can be used to register users and add feeds to be aggregated. Several of the commands will operate in the [current_user_name] defined in the config file.

### Register
`gator register <username>`

Register will add a new user to the database so their feeds can be tracked and set the user as the current active user in the config.

### Login
`gator login <username>`

Login finds the user in the database and sets it as the current active user in the config. It will return an error if the user is not registered.

### Users
`gator users`

Users will print the list of users in the database.

### Add Feed
`gator addfeed <name> <url>`

Add Feed will add a feed to the database with the given *name* and *url*. The name and URL must be unique. If the feed is added successfully, the active user will follow the feed.

### Feeds
`gator feeds`

Feeds displays all the feeds currently stored in the database.

### Follow
`gator follow <url>`

Follow adds the active user to follow the feed with the given *url*.

### Unfollow
`gator unfollow <url>`

Unfollow removes the active user from following the feed with the given *url*.

### Following
`gator following`

Following displays all the feeds the active user is currently following.

### Agg
`gator agg <time_interval>`

Agg is a long-running program designed to run in the background to continually scrape feeds that have been added to the database and update their posts. The *time_interval* defines how frequently the loop will process a feed, with a format like *1m* or *30s*. Use CTRL-C to end the program.

### Browse
`gator browse [limit]`

Browse will display the latest posts from the active user's feeds. If the optional *limit* parameter is omitted, it will default to 2 posts.