# Stages:

## Respond to multiple PINGs

```
echo -e "PING\nPING" | redis-cli

PONG
PONG
```

## Handle conncurrent clients

```
redis-cli ping & redis-cli ping & redis-cli
```

## echo

```
redis-cli echo "Hello World"
```

## Set and Get

```
redis-cli set mykey "Hello World"
redis-cli get mykey
```

## Set with expiry

```
redis-cli set mykey "Hello World" px 100
```

> px is in milliseconds

## start on specific port

```
./your_program.sh --port 5000
redis-cli -p 5000 ping
```

## start follower and set leader addr

```
./your_program.sh --port 5000 --replicaof "0.0.0.0 6379"
```

## replication test

```
redis-cli set foo 123
redis-cli set bar 456
redis-cli set baz 789

redis-cli get foo
redis-cli get bar
```

## incr

```
redis-cli SET foo 100
redis-cli INCR foo
redis-cli INCR foo
```

# transactions

```
redis-cli multi
redis-cli exec
> (empty array)

redis-cli exec
> (error) ERR EXEC without MULTI

redis-cli multi
redis-cli set foo bar
redis-cli get foo
> (nil)

redis-cli exec
redis-cli get foo
> "bar"
```

# xrange

```
redis-cli XADD some_key 1526985054069-0 temperature 36 humidity 95
redis-cli XADD some_key 1526985054079-0 temperature 37 humidity 94
redis-cli XRANGE some_key 1526985054069 1526985054079
```

```
redis-cli XADD stream_key 0-1 foo bar
redis-cli XADD stream_key 0-2 bar baz
redis-cli XADD stream_key 0-3 baz foo
redis-cli XRANGE stream_key 0-2 0-3
```

```
redis-cli xadd grape 0-1 foo bar
redis-cli xadd grape 0-2 foo bar
redis-cli xadd grape 0-3 foo bar
redis-cli xrange grape - 0-2
```

# xread

```
redis-cli XADD some_key 1526985054069-0 temperature 36 humidity 95
redis-cli XADD some_key 1526985054079-0 temperature 37 humidity 94
redis-cli XREAD streams some_key 1526985054069-0
```

```
redis-cli XADD stream_key 0-1 temperature 95
redis-cli XADD other_stream_key 0-2 humidity 97
redis-cli XREAD streams stream_key other_stream_key 0-0 0-1
```
