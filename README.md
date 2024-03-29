# filewatcher

Filewatcher consists of a server and client. The client will monitor all files in a given path and synchronize it to the server on connect and as they change.

We also do some checks of the actual data to not send a file the remote already has over again. We try to only send the deltas of what has changed when possible to save bandwidth and transfer time.

Developed and tested on Linux, but might work on other OS'es with a functioning Go compiler.

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
./filewatcher receive <target-path> <listen-port>

# Example:
mkdir /tmp/syncdir && \
./filewatcher receive /tmp/syncdir 9090
```

### Sender
```bash
./filewatcher sync <path-to-sync> <remote-host> <port>

# Example:
./filewatcher sync . 127.0.0.1 9090
```


## Suggested improvements
* Authentication
* TLS
* Compression
* Better error handling for edgecases, etc
* Tests
* Improved code quality