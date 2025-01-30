# CreateMod.com

This repository contains all the files needed to run CreateMod.com

To run make sure you have go installed locally and run:
```
go run ./cmd/server/main.go serve
```

## Migration from Wordpress

CreateMod.com was originally build as a Wordpress site and this project will migrate from that to a Go backend.

migration_data folder should contain images and schematics from the base url path, so /wp-content/folder/image.png

dbdata should contain a full database dump for testing the migration

## Docker

[Docker](https://www.docker.com/) is provided to make local development easier. Use the following command after ensuring that [Docker](https://www.docker.com/) is installed:

```
docker compose up
```

The application will then be available at [http://127.0.0.1:8090](http://127.0.0.1:8090)

The build parameter should be used to rebuild the containers if you are making changes in the Go code.

```
docker compose up --build
```

## Environmental Variables

### MySQL

MySQL variables can be set if you need to migrate from an existing Wordpress database otherwise these can be left blank

```
MYSQL_HOST=localhost:3306
MYSQL_DB=webapp
MYSQL_USER=webapp
MYSQL_PASS=root
```

### Auto Migrate

Auto Migrate can be used to automatically generate database migration files when changes to the data structures are made.

```
AUTO_MIGRATE=true
```

### Create Admin

If Create Admin is set to true an admin is generated. This is convenient for local development.

```
CREATE_ADMIN=true
```

The default credentials are `local@createmod.com` and `jfq.utb*jda2abg!WCR`. Do not use these credentials in a live environment.

### Dummy data

You can set the following to true to generate dummy data. Please note that it will only work when running the migrations for the first time. Delete the `pb_data` to reset. WARNING this deletes all data.

```
DUMMY_DATA=true
```