# CreateMod.com

This repository contains all the files needed to run CreateMod.com

To run make sure you have go installed locally and run:
```
go run .\cmd\server\main.go serve
```

## Docker

[Docker](https://www.docker.com/) is provided to make local development easier. Use the following command after ensuring that [Docker](https://www.docker.com/) is installed:

```
docker-compose up
```

The application will then be available at [http://127.0.0.1:8090](http://127.0.0.1:8090)

The build parameter should be used to rebuild the containers if you are making changes in the Go code.

```
docker-compose build --build
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