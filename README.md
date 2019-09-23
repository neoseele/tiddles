# Tiddles

A ship'cat wonders around, and catching mices, or bugs. :)

## Build the image

* build with Cloud Builder

  ```sh
  make build
  ```

* build locally

  ```sh
  make build-local
  ```

## Run the app locally

```sh
docker run --rm -d -p 127.0.0.1:80:80 -p 127.0.0.1:443:443 --name debug tiddles:local
```

## Clean up

```sh
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

* `/people`
  * `GET /people`
  * `GET /people/{id}`
  * `POST /people/{id}`
  * `PUT /people/{id}`
  * `DELETE /people/{id}`
  > sample rest API

* `/dump`
