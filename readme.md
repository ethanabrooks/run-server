## Dependencies
- [docker](https://docs.docker.com/get-docker/)
- [postgres](https://www.postgresql.org/download/)
- [go](https://golang.org/doc/install)

## Usage
Start `postgres`:
```shell_session
docker run --name some-postgres -e POSTGRES_PASSWORD=mysecretpassword -d --rm --publish 5432:5432 postgres
```
Set `postgres` password environment variable:
```shell_session
export PGPASSWORD=mysecretpassword
```
Create database (don't worry, this will do nothing if you have already run it):
```shell_session
cat create.sql | psql -U postgres -p 5432 -h localhost 
```
Run tests:
```shell_session
docker-compose -f test.yml up --build
```
Make sure to run
```shell_session
docker-compose down --remove-orphans 
```
after each test or subsequent tests will fail.

To destroy the database:
```shell_session
cat drop.sql | psql -U postgres -p 5432 -h localhost 
```
