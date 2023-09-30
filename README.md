# Go HTTP server to count requests that it has received during previous 60 seconds.

## API
#### POST /requests

Accepts no parameters, returns number of requests recieved by a server during previous 60 seconds, **without current request**. Example:
```bash
curl "127.0.0.1:8080/requests"
Request count in the last minute: 1
```

## Application design

##### Application consists of 3 parts, running in separate goroutines:
- API server
- Job for persisting logged data into a file
- Job for removing older records from file

##### Implementation details
- File dumps are done asynchronously to keep response time low.
- Concurrent access to data structure and file is managed by two separate mutexes.
- Errors at every stage of running except for starting the server are considered not fatal and are only processed by logging.
- Unit and integration tests are present. Tests are not comprehensive, they are meant to give an example of what tests for this service could look like.

##### Possible improvements
- Depending on the number of requests in a minute, reading from a backup file in one chunk can load memory too much. If that is the case, streaming data should be implemented.
- Parameters like port and number of seconds in a moving window can be moved to a configuration file or read from environmental variables for convenience.

## How to run
```bash
docker build -t coding_task .
docker run -p 8080:8080 --rm coding_task
```
