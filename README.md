# Tiddles

A ship'cat wonders around catching mices, or bugs. :)

## Build the image

* build with Cloud Builder

  ```sh
  make build
  ```

* build locally

  ```sh
  make build-local
  ```

## Build and Deploy to GKE with `skaffold`

```sh
make build-deploy
```

## Run the app locally

```sh
make run-server
make run-client
```

## Clean up

```sh
make stop
make clean
```

## Debug

```sh
# run
docker run --rm -d -p 127.0.0.1:80:80 -p 127.0.0.1:443:443 --name debug tiddles:local

# stop
docker stop debug
docker rmi tiddles:local
```

## Rest APIs

* `GET /`
* `GET /error`
* `GET /health`
* `GET /liveness`
* `GET /readiness`
* `GET /ping`
* `GET /ping-backend`
* `GET /ping-backend-with-db`
* `GET /ping-grpc-backend`

* `/stress`
  * `GET /stress/cpu`
  * `GET /stress/cpu?load=0.1&duration=10`
  > load: push cpu load to 0.1; duration: keep the cpu load for 10 seconds
  * `GET /stress/memory`
  * `GET /stress/memory?size=100`
  > size: allocate memroy in MB

* `/dns`
  * `GET /dns/weight?=1000`
  > weight: number of concurrent dns queries in each web request

* `/db`
  * `GET /db`
  * `GET /db/{id}`
  * `POST /db/{id}`
  * `PUT /db/{id}`
  * `DELETE /db/{id}`
  > sample rest API

* `/dump`
