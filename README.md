# filewatcher

Filewatcher consists of a server and client. The client will monitor all files in a given path and synchronize it to the server as they change.

We also do some checks of the actual data to not send a file the remote already has over again. We also try to only send the deltas of what has changed when possible to save bandwidth and transfer time.

## How to build
Standalone binary
```bash
make
./filewatcher
```

Docker image
```bash
make docker
docker run --net=host --tm -it <extra args> filewatcher
```

## How to use

### Receiver
```bash
./filewatcher receive <target-path>
```

### Sender
```bash
./filewatcher send <path-to-sync>
```


## Suggested improvements
* Authentication
* TLS
* Compression
* Tests
* Improved code quality