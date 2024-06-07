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